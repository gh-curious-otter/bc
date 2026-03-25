// Package slackgw implements the gateway.Adapter for Slack.
package slackgw

import (
	"context"
	"fmt"
	"sync"

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
	botToken  string
	appToken  string
	onMessage func(gateway.InboundMessage)
	botUserID string
	chatMu    sync.RWMutex
	// channelMap maps channel_id → channel name
	channelMap map[string]string
}

var _ gateway.Adapter = (*Adapter)(nil)

// New creates a new Slack adapter using Socket Mode.
func New(botToken, appToken string) *Adapter {
	return &Adapter{
		botToken:   botToken,
		appToken:   appToken,
		channelMap: make(map[string]string),
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

	// Run socket mode (blocks until context cancelled)
	return sm.RunContext(ctx)
}

func (a *Adapter) Stop(_ context.Context) error {
	// Socket mode client stops when context is cancelled in Start
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

func (a *Adapter) Health(_ context.Context) error {
	if a.api == nil {
		return fmt.Errorf("slack: not connected")
	}
	return nil
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
	// Skip bot messages and message edits/deletes
	if ev.User == a.botUserID || ev.User == "" || ev.SubType != "" {
		return
	}

	content := ev.Text
	if content == "" {
		return
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

	// Resolve user name
	sender := ev.User
	if a.api != nil {
		if userInfo, err := a.api.GetUserInfo(ev.User); err == nil {
			sender = userInfo.RealName
			if sender == "" {
				sender = userInfo.Name
			}
		}
	}

	msg := gateway.InboundMessage{
		ChannelID:   ev.Channel,
		ChannelName: channelName,
		Sender:      sender,
		SenderID:    ev.User,
		Content:     content,
		MessageID:   ev.TimeStamp,
	}

	log.Info("slack: received message",
		"channel", channelName,
		"sender", sender,
		"content", truncate(content, 50))

	if a.onMessage != nil {
		a.onMessage(msg)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
