// Package channel - query.go provides query APIs for message history.
package channel

import (
	"strings"
	"time"
)

// QueryOptions configures message query parameters.
type QueryOptions struct {
	Before *time.Time // Messages before this time
	After  *time.Time // Messages after this time
	Sender string     // Filter by sender
	Limit  int        // Max messages to return (default: 50, max: 100)
	Offset int        // Skip first N messages (for pagination)
}

// DefaultQueryOptions returns sensible defaults.
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		Limit: 50,
	}
}

// QueryResult contains paginated query results.
type QueryResult struct {
	Messages   []HistoryEntry `json:"messages"`
	Total      int            `json:"total"`       // Total matching messages
	HasMore    bool           `json:"has_more"`    // More messages available
	NextOffset int            `json:"next_offset"` // Offset for next page
}

// Query returns messages matching the given options.
func (s *Store) Query(channelName string, opts QueryOptions) (*QueryResult, error) {
	// Apply defaults
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Fetch history from backend
	history, err := s.GetHistory(channelName)
	if err != nil {
		return nil, err
	}

	// Filter messages
	filtered := make([]HistoryEntry, 0)
	for _, entry := range history {
		if !matchesQuery(entry, opts) {
			continue
		}
		filtered = append(filtered, entry)
	}

	total := len(filtered)

	// Apply offset
	if opts.Offset > 0 {
		if opts.Offset >= len(filtered) {
			filtered = []HistoryEntry{}
		} else {
			filtered = filtered[opts.Offset:]
		}
	}

	// Apply limit
	hasMore := false
	if len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
		hasMore = true
	}

	return &QueryResult{
		Messages:   filtered,
		Total:      total,
		HasMore:    hasMore,
		NextOffset: opts.Offset + len(filtered),
	}, nil
}

// matchesQuery checks if a history entry matches query options.
func matchesQuery(entry HistoryEntry, opts QueryOptions) bool {
	// Time filters
	if opts.Before != nil && !entry.Time.Before(*opts.Before) {
		return false
	}
	if opts.After != nil && !entry.Time.After(*opts.After) {
		return false
	}

	// Sender filter
	if opts.Sender != "" && entry.Sender != opts.Sender {
		return false
	}

	return true
}

// SearchOptions configures full-text search.
type SearchOptions struct {
	Since    *time.Time // Only messages after this time
	Channels []string   // Limit to specific channels (empty = all)
	Limit    int        // Max results (default: 50)
}

// SearchResult contains search results.
type SearchResult struct {
	Channel string       `json:"channel"`
	Entry   HistoryEntry `json:"entry"`
}

// Search performs a simple text search across channels.
// Note: This is a basic implementation. For production use with large
// datasets, use the SQLite FTS5 backend when available.
func (s *Store) Search(query string, opts SearchOptions) ([]SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	query = strings.ToLower(query)
	results := make([]SearchResult, 0)

	// Get channel list from backend
	channels := s.List()

	for _, ch := range channels {
		// Check if channel is in filter list
		if len(opts.Channels) > 0 {
			found := false
			for _, c := range opts.Channels {
				if c == ch.Name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		history, err := s.GetHistory(ch.Name)
		if err != nil {
			continue
		}

		for _, entry := range history {
			// Time filter
			if opts.Since != nil && !entry.Time.After(*opts.Since) {
				continue
			}

			// Simple text match
			if strings.Contains(strings.ToLower(entry.Message), query) ||
				strings.Contains(strings.ToLower(entry.Sender), query) {
				results = append(results, SearchResult{
					Channel: ch.Name,
					Entry:   entry,
				})

				if len(results) >= opts.Limit {
					return results, nil
				}
			}
		}
	}

	return results, nil
}

// GetMentions returns messages mentioning an agent.
func (s *Store) GetMentions(agent string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	mention := "@" + agent
	results := make([]SearchResult, 0)

	channels := s.List()
	for _, ch := range channels {
		history, err := s.GetHistory(ch.Name)
		if err != nil {
			continue
		}

		for _, entry := range history {
			if strings.Contains(entry.Message, mention) {
				results = append(results, SearchResult{
					Channel: ch.Name,
					Entry:   entry,
				})

				if len(results) >= limit {
					return results, nil
				}
			}
		}
	}

	return results, nil
}
