// Package discord implements the gateway.Adapter for Discord.
package discord

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/gh-curious-otter/bc/pkg/gateway"
	"github.com/gh-curious-otter/bc/pkg/log"
)

// Adapter implements gateway.Adapter for Discord.
type Adapter struct {
	session   *discordgo.Session
	onMessage func(gateway.InboundMessage)
	// guildChannels maps channel_id → channel name
	guildChannels map[string]string
	token         string
	// guildIDs tracks guilds the bot is in
	guildIDs      []string
	connected     bool
	lastMessageAt time.Time
	lastError     string
	chatMu        sync.RWMutex
}

var _ gateway.Adapter = (*Adapter)(nil)
var _ gateway.StatusReporter = (*Adapter)(nil)

// New creates a new Discord adapter.
func New(token string) *Adapter {
	return &Adapter{
		token:         token,
		guildChannels: make(map[string]string),
	}
}

func (a *Adapter) Name() string { return "discord" }

func (a *Adapter) Start(ctx context.Context, onMessage func(gateway.InboundMessage)) error {
	a.onMessage = onMessage

	session, err := discordgo.New("Bot " + a.token)
	if err != nil {
		return fmt.Errorf("discord: failed to create session: %w", err)
	}
	a.session = session

	// Set intents: we need guild messages and message content
	session.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildMessages |
		discordgo.IntentMessageContent

	// Register message handler
	session.AddHandler(a.handleMessage)

	// Register ready handler to discover guilds/channels
	session.AddHandler(a.handleReady)

	if err := session.Open(); err != nil {
		return fmt.Errorf("discord: failed to connect: %w", err)
	}
	log.Info("discord: connected", "bot", session.State.User.Username)

	// Block until context is canceled
	<-ctx.Done()
	return session.Close()
}

func (a *Adapter) Stop(_ context.Context) error {
	if a.session != nil {
		return a.session.Close()
	}
	return nil
}

func (a *Adapter) Send(_ context.Context, channelID, sender, content string) error {
	if a.session == nil {
		return fmt.Errorf("discord: not connected")
	}

	// Format: **agent_name**: message
	text := fmt.Sprintf("**%s**: %s", sender, content)

	if _, err := a.session.ChannelMessageSend(channelID, text); err != nil {
		return fmt.Errorf("discord: send failed: %w", err)
	}

	log.Info("discord: sent message", "channel_id", channelID, "sender", sender)
	return nil
}

func (a *Adapter) Channels(_ context.Context) ([]gateway.ExternalChannel, error) {
	a.chatMu.RLock()
	defer a.chatMu.RUnlock()

	channels := make([]gateway.ExternalChannel, 0, len(a.guildChannels))
	for id, name := range a.guildChannels {
		channels = append(channels, gateway.ExternalChannel{
			ID:   id,
			Name: name,
			Type: "channel",
		})
	}
	return channels, nil
}

func (a *Adapter) Health(_ context.Context) error {
	if a.session == nil {
		return fmt.Errorf("discord: not connected")
	}
	// Live probe: check session state
	if a.session.State == nil || a.session.State.User == nil {
		a.chatMu.Lock()
		a.connected = false
		a.lastError = "session not ready"
		a.chatMu.Unlock()
		return fmt.Errorf("discord: session not ready")
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

// handleReady processes the Ready event to discover guilds and channels.
func (a *Adapter) handleReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Info("discord: ready", "guilds", len(r.Guilds))

	for _, guild := range r.Guilds {
		a.guildIDs = append(a.guildIDs, guild.ID)

		channels, err := a.session.GuildChannels(guild.ID)
		if err != nil {
			log.Warn("discord: failed to list channels", "guild", guild.ID, "error", err)
			continue
		}

		a.chatMu.Lock()
		for _, ch := range channels {
			// Only text channels
			if ch.Type == discordgo.ChannelTypeGuildText {
				a.guildChannels[ch.ID] = ch.Name
				log.Info("discord: discovered channel", "channel", ch.Name, "id", ch.ID, "guild", guild.ID)
			}
		}
		a.chatMu.Unlock()
	}
}

// handleMessage processes incoming Discord messages.
func (a *Adapter) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Skip bot messages
	if m.Author.Bot {
		return
	}

	content := m.Content
	if content == "" {
		return
	}

	// Get channel name
	a.chatMu.RLock()
	channelName, ok := a.guildChannels[m.ChannelID]
	a.chatMu.RUnlock()
	if !ok {
		channelName = m.ChannelID
	}

	sender := m.Author.Username
	if m.Member != nil && m.Member.Nick != "" {
		sender = m.Member.Nick
	}

	msg := gateway.InboundMessage{
		ChannelID:   m.ChannelID,
		ChannelName: channelName,
		Sender:      sender,
		SenderID:    m.Author.ID,
		Content:     content,
		MessageID:   m.ID,
		Timestamp:   m.Timestamp,
	}

	log.Info("discord: received message",
		"channel", channelName,
		"sender", sender,
		"content", gateway.Truncate(content, 50))

	if a.onMessage != nil {
		a.onMessage(msg)
	}
}
