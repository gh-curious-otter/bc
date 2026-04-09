// Package slackgw implements the gateway.Adapter for Slack.
package slackgw

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/gh-curious-otter/bc/pkg/gateway"
	"github.com/gh-curious-otter/bc/pkg/log"
)

// Adapter implements gateway.Adapter for Slack using Socket Mode.
type Adapter struct {
	api       *slack.Client
	sm        *socketmode.Client
	onMessage func(gateway.InboundMessage)
	// channelMap maps channel_id → channel name
	channelMap map[string]string
	// userCache maps user_id → display name
	userCache     map[string]string
	botToken      string
	appToken      string
	botUserID     string
	connected     bool
	lastMessageAt time.Time
	lastError     string
	chatMu        sync.RWMutex
}

var _ gateway.Adapter = (*Adapter)(nil)
var _ gateway.FileSender = (*Adapter)(nil)
var _ gateway.StatusReporter = (*Adapter)(nil)

// New creates a new Slack adapter using Socket Mode.
func New(botToken, appToken string) *Adapter {
	return &Adapter{
		botToken:   botToken,
		appToken:   appToken,
		channelMap: make(map[string]string),
		userCache:  make(map[string]string),
	}
}

func (a *Adapter) Name() string { return "slack" }

func (a *Adapter) Start(ctx context.Context, onMessage func(gateway.InboundMessage)) error {
	a.onMessage = onMessage

	api := slack.New(
		a.botToken,
		slack.OptionAppLevelToken(a.appToken),
	)
	a.api = api

	// Get bot user ID
	authResp, err := api.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("slack: auth test failed: %w", err)
	}
	a.botUserID = authResp.UserID
	a.chatMu.Lock()
	a.connected = true
	a.chatMu.Unlock()
	log.Info("slack: connected", "bot_user_id", a.botUserID, "team", authResp.Team)

	// Discover channels the bot is in
	if err := a.discoverChannels(ctx); err != nil {
		log.Warn("slack: channel discovery failed", "error", err)
	}

	// Create socket mode client
	sm := socketmode.New(api)
	a.sm = sm

	// Handle events in a goroutine
	go a.handleEvents(ctx, sm)

	// Run socket mode (blocks until context canceled)
	return sm.RunContext(ctx)
}

func (a *Adapter) Stop(_ context.Context) error {
	// Socket mode client stops when context is canceled in Start
	return nil
}

func (a *Adapter) Send(ctx context.Context, channelID, sender, content string) error {
	if a.api == nil {
		return fmt.Errorf("slack: not connected")
	}

	// Use chat:write.customize to show agent name
	_, _, err := a.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionText(content, false),
		slack.MsgOptionUsername(sender),
		slack.MsgOptionIconEmoji(":robot_face:"),
	)
	if err != nil {
		return fmt.Errorf("slack: send failed: %w", err)
	}

	log.Info("slack: sent message", "channel_id", channelID, "sender", sender)
	return nil
}

// SendFile uploads a file to a Slack channel.
func (a *Adapter) SendFile(ctx context.Context, channelID, sender, filename string, data []byte, mimeType string) error {
	if a.api == nil {
		return fmt.Errorf("slack: not connected")
	}

	_, err := a.api.UploadFileContext(ctx, slack.UploadFileParameters{
		Filename:       filename,
		Reader:         bytes.NewReader(data),
		FileSize:       len(data),
		Channel:        channelID,
		Title:          fmt.Sprintf("%s: %s", sender, filename),
		InitialComment: fmt.Sprintf("Shared by %s", sender),
	})
	if err != nil {
		return fmt.Errorf("slack: file upload failed: %w", err)
	}

	log.Info("slack: uploaded file", "channel_id", channelID, "sender", sender, "filename", filename)
	return nil
}

func (a *Adapter) Channels(_ context.Context) ([]gateway.ExternalChannel, error) {
	a.chatMu.RLock()
	defer a.chatMu.RUnlock()

	channels := make([]gateway.ExternalChannel, 0, len(a.channelMap))
	for id, name := range a.channelMap {
		channels = append(channels, gateway.ExternalChannel{
			ID:   id,
			Name: name,
			Type: "channel",
		})
	}
	return channels, nil
}

func (a *Adapter) Health(ctx context.Context) error {
	if a.api == nil {
		return fmt.Errorf("slack: not connected")
	}
	// Live probe: call auth.test to verify the connection
	if _, err := a.api.AuthTestContext(ctx); err != nil {
		a.chatMu.Lock()
		a.connected = false
		a.lastError = err.Error()
		a.chatMu.Unlock()
		return fmt.Errorf("slack: auth test failed: %w", err)
	}
	a.chatMu.Lock()
	a.connected = true
	a.lastError = ""
	a.chatMu.Unlock()
	return nil
}

// Status returns the current connection state.
func (a *Adapter) Status() gateway.AdapterStatus {
	a.chatMu.RLock()
	defer a.chatMu.RUnlock()
	return gateway.AdapterStatus{
		Connected:     a.connected,
		LastMessageAt: a.lastMessageAt,
		Error:         a.lastError,
	}
}

// discoverChannels lists channels the bot is a member of.
func (a *Adapter) discoverChannels(ctx context.Context) error {
	// Use GetConversationsForUser to list channels the bot is in
	params := &slack.GetConversationsForUserParameters{
		UserID:          a.botUserID,
		Types:           []string{"public_channel"},
		ExcludeArchived: true,
		Limit:           200,
	}

	channels, _, err := a.api.GetConversationsForUserContext(ctx, params)
	if err != nil {
		// Fallback: try listing all public channels
		log.Warn("slack: GetConversationsForUser failed, trying GetConversations", "error", err)
		listParams := &slack.GetConversationsParameters{
			Types:           []string{"public_channel"},
			ExcludeArchived: true,
			Limit:           200,
		}
		channels, _, err = a.api.GetConversationsContext(ctx, listParams)
		if err != nil {
			return fmt.Errorf("list conversations: %w", err)
		}
	}

	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	for _, ch := range channels {
		if ch.IsMember {
			a.channelMap[ch.ID] = ch.Name
			log.Info("slack: discovered channel", "channel", ch.Name, "id", ch.ID)
		}
	}
	return nil
}

// handleEvents processes Socket Mode events.
func (a *Adapter) handleEvents(ctx context.Context, sm *socketmode.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-sm.Events:
			if !ok {
				return
			}
			a.processEvent(sm, evt)
		}
	}
}

// processEvent handles a single Socket Mode event.
func (a *Adapter) processEvent(sm *socketmode.Client, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			log.Warn("slack: failed to cast EventsAPI event")
			return
		}
		sm.Ack(*evt.Request)
		a.handleEventsAPI(eventsAPIEvent)

	case socketmode.EventTypeConnecting:
		log.Info("slack: connecting via Socket Mode...")

	case socketmode.EventTypeConnected:
		log.Info("slack: Socket Mode connected")

	case socketmode.EventTypeConnectionError:
		log.Warn("slack: Socket Mode connection error")

	case socketmode.EventTypeHello:
		log.Info("slack: Socket Mode hello received")

	default:
		log.Info("slack: unhandled event type", "type", evt.Type)
		// Acknowledge unknown events to prevent retries
		if evt.Request != nil {
			sm.Ack(*evt.Request)
		}
	}
}

// handleEventsAPI processes Events API payloads.
func (a *Adapter) handleEventsAPI(event slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			a.handleMessageEvent(ev)
		}
	}
}

// handleMessageEvent processes a single message event.
func (a *Adapter) handleMessageEvent(ev *slackevents.MessageEvent) {
	// Skip bot messages and message edits/deletes.
	// Allow file_share subtype for image sharing.
	if ev.User == a.botUserID || ev.User == "" {
		return
	}
	if ev.SubType != "" && ev.SubType != "file_share" {
		return
	}

	content := ev.Text
	if content == "" && ev.SubType != "file_share" {
		return
	}
	// For file_share events with no text, add a descriptive message
	if content == "" && ev.SubType == "file_share" {
		content = "[shared a file]"
	}

	// Resolve channel name — try cache first, then API lookup
	a.chatMu.RLock()
	channelName, ok := a.channelMap[ev.Channel]
	a.chatMu.RUnlock()
	if !ok {
		// Lookup via API and cache the result
		if a.api != nil {
			if chInfo, err := a.api.GetConversationInfo(&slack.GetConversationInfoInput{
				ChannelID: ev.Channel,
			}); err == nil && chInfo != nil {
				channelName = chInfo.Name
				a.chatMu.Lock()
				a.channelMap[ev.Channel] = channelName
				a.chatMu.Unlock()
			}
		}
		if channelName == "" {
			channelName = ev.Channel
		}
	}

	// Resolve user name — cache lookups to avoid repeated API calls
	sender := ev.User
	if a.api != nil {
		a.chatMu.RLock()
		cachedName, cached := a.userCache[ev.User]
		a.chatMu.RUnlock()
		if cached {
			sender = cachedName
		} else if userInfo, err := a.api.GetUserInfo(ev.User); err == nil {
			sender = userInfo.RealName
			if sender == "" {
				sender = userInfo.Name
			}
			a.chatMu.Lock()
			if a.userCache == nil {
				a.userCache = make(map[string]string)
			}
			a.userCache[ev.User] = sender
			a.chatMu.Unlock()
		} else {
			log.Warn("slack: failed to resolve user", "user_id", ev.User, "error", err)
		}
	}

	now := time.Now()
	msg := gateway.InboundMessage{
		Timestamp:   now,
		ChannelID:   ev.Channel,
		ChannelName: channelName,
		Sender:      sender,
		SenderID:    ev.User,
		Content:     content,
		MessageID:   ev.TimeStamp,
	}

	a.chatMu.Lock()
	a.lastMessageAt = now
	a.chatMu.Unlock()

	log.Info("slack: received message",
		"channel", channelName,
		"sender", sender,
		"content", gateway.Truncate(content, 50))

	if a.onMessage != nil {
		a.onMessage(msg)
	}
}
