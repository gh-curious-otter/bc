package gateway

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gh-curious-otter/bc/pkg/log"
)

// Manager orchestrates all gateway adapters and routes messages.
type Manager struct {
	adapters map[string]Adapter
	// channelMap maps "telegram:<group_name>" → channelRoute
	channelMap map[string]channelRoute
	// onInbound is called when a message arrives from an external platform.
	// Typically wired to ChannelService.Send + SSE hub.
	onInbound func(bcChannel, sender, content string)
	mu        sync.RWMutex
}

type channelRoute struct {
	Platform  string
	ChannelID string
	Adapter   Adapter
}

// NewManager creates a new gateway manager.
func NewManager() *Manager {
	return &Manager{
		adapters:   make(map[string]Adapter),
		channelMap: make(map[string]channelRoute),
	}
}

// SetInboundHandler sets the callback for inbound messages from external platforms.
func (m *Manager) SetInboundHandler(fn func(bcChannel, sender, content string)) {
	m.onInbound = fn
}

// Register adds an adapter to the manager.
func (m *Manager) Register(adapter Adapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adapters[adapter.Name()] = adapter
}

// Start discovers channels from all adapters and begins receiving messages.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	adapters := make([]Adapter, 0, len(m.adapters))
	for _, a := range m.adapters {
		adapters = append(adapters, a)
	}
	m.mu.RUnlock()

	// Discover channels from each adapter
	for _, a := range adapters {
		channels, err := a.Channels(ctx)
		if err != nil {
			log.Warn("gateway: failed to discover channels", "adapter", a.Name(), "error", err)
			continue
		}
		m.mu.Lock()
		for _, ch := range channels {
			bcName := a.Name() + ":" + sanitizeChannelName(ch.Name)
			m.channelMap[bcName] = channelRoute{
				Platform:  a.Name(),
				ChannelID: ch.ID,
				Adapter:   a,
			}
			log.Info("gateway: discovered channel", "bc_channel", bcName, "platform_id", ch.ID)
		}
		m.mu.Unlock()
	}

	// Start all adapters in goroutines
	var wg sync.WaitGroup
	for _, a := range adapters {
		wg.Add(1)
		go func(adapter Adapter) {
			defer wg.Done()
			if err := adapter.Start(ctx, m.handleInbound); err != nil && ctx.Err() == nil {
				log.Error("gateway: adapter stopped with error", "adapter", adapter.Name(), "error", err)
			}
		}(a)
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

// Stop gracefully shuts down all adapters.
func (m *Manager) Stop(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.adapters {
		if err := a.Stop(ctx); err != nil {
			log.Warn("gateway: stop error", "adapter", a.Name(), "error", err)
		}
	}
}

// Send routes a message from a bc channel to the appropriate external platform.
// Returns true if the channel is an external gateway channel and was handled.
func (m *Manager) Send(ctx context.Context, bcChannel, sender, content string) (bool, error) {
	m.mu.RLock()
	route, ok := m.channelMap[bcChannel]
	m.mu.RUnlock()
	if !ok {
		return false, nil // not a gateway channel
	}

	if err := route.Adapter.Send(ctx, route.ChannelID, sender, content); err != nil {
		return true, fmt.Errorf("gateway send to %s: %w", bcChannel, err)
	}
	return true, nil
}

// IsGatewayChannel returns true if the channel name belongs to an external gateway.
func (m *Manager) IsGatewayChannel(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.channelMap[name]
	return ok
}

// ExternalChannels returns all discovered external channels.
func (m *Manager) ExternalChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.channelMap))
	for name := range m.channelMap {
		names = append(names, name)
	}
	return names
}

// handleInbound processes a message from an external platform into bc.
func (m *Manager) handleInbound(msg InboundMessage) {
	// Build the bc channel name: "telegram:group_name"
	// Find which adapter this came from
	m.mu.RLock()
	var bcChannel string
	for name, route := range m.channelMap {
		if route.ChannelID == msg.ChannelID {
			bcChannel = name
			break
		}
	}
	m.mu.RUnlock()

	if bcChannel == "" {
		// Channel not mapped yet — try to add it dynamically
		// Use the channel name from the message
		if msg.ChannelName != "" {
			for _, a := range m.adapters {
				bcChannel = a.Name() + ":" + sanitizeChannelName(msg.ChannelName)
				m.mu.Lock()
				m.channelMap[bcChannel] = channelRoute{
					Platform:  a.Name(),
					ChannelID: msg.ChannelID,
					Adapter:   a,
				}
				m.mu.Unlock()
				log.Info("gateway: dynamically mapped channel", "bc_channel", bcChannel, "platform_id", msg.ChannelID)
				break
			}
		}
		if bcChannel == "" {
			log.Warn("gateway: unmapped inbound message", "channel_id", msg.ChannelID)
			return
		}
	}

	sender := fmt.Sprintf("[telegram] %s", msg.Sender)
	if m.onInbound != nil {
		m.onInbound(bcChannel, sender, msg.Content)
	}
}

// sanitizeChannelName converts a group name to a valid bc channel name.
func sanitizeChannelName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	// Remove any characters that aren't alphanumeric, dash, or underscore
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
