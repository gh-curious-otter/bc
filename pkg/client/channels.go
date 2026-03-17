package client

import "context"

// ChannelsClient provides channel operations via the daemon.
type ChannelsClient struct {
	client *Client
}

// ChannelInfo represents channel data returned by the daemon.
type ChannelInfo struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// List returns all channels from the daemon.
func (ch *ChannelsClient) List(ctx context.Context) ([]ChannelInfo, error) {
	var channels []ChannelInfo
	if err := ch.client.get(ctx, "/api/channels", &channels); err != nil {
		return nil, err
	}
	return channels, nil
}
