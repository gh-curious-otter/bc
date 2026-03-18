package channel

// ChannelBackend is the storage interface implemented by both SQLiteStore and PostgresStore.
// It covers the full set of operations used by the higher-level Store wrapper.
type ChannelBackend interface {
	// Lifecycle
	Close() error

	// Channel operations
	CreateChannel(name string, channelType ChannelType, description string) (*ChannelInfo, error)
	GetChannel(name string) (*ChannelInfo, error)
	ListChannels() ([]*ChannelInfo, error)
	DeleteChannel(name string) error
	SetChannelDescription(channelName, description string) error

	// Member operations
	AddMember(channelName, agentID string) error
	RemoveMember(channelName, agentID string) error
	GetMembers(channelName string) ([]string, error)

	// Message operations
	AddMessage(channelName, sender, content string, msgType MessageType, metadata string) (*Message, error)
	GetHistory(channelName string, limit int) ([]*Message, error)

	// Reaction operations
	ToggleReaction(messageID int64, emoji, userID string) (bool, error)
	GetReactions(messageID int64) (map[string][]string, error)
}
