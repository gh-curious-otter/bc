// Package channel - query.go provides query APIs for message history.
package channel

import (
	"fmt"
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	// Apply defaults
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Filter messages
	filtered := make([]HistoryEntry, 0)
	for _, entry := range ch.History {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	query = strings.ToLower(query)
	results := make([]SearchResult, 0)

	for name, ch := range s.channels {
		// Check if channel is in filter list
		if len(opts.Channels) > 0 {
			found := false
			for _, c := range opts.Channels {
				if c == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		for _, entry := range ch.History {
			// Time filter
			if opts.Since != nil && !entry.Time.After(*opts.Since) {
				continue
			}

			// Simple text match
			if strings.Contains(strings.ToLower(entry.Message), query) ||
				strings.Contains(strings.ToLower(entry.Sender), query) {
				results = append(results, SearchResult{
					Channel: name,
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	mention := "@" + agent
	results := make([]SearchResult, 0)

	for name, ch := range s.channels {
		for _, entry := range ch.History {
			if strings.Contains(entry.Message, mention) {
				results = append(results, SearchResult{
					Channel: name,
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
