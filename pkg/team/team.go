// Package team provides team management for bc.
// Teams are organizational units that group agents together.
package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Team represents an organizational team of agents.
//
//nolint:govet // fieldalignment: minor 8 byte difference not worth reordering for JSON readability
type Team struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Members     []string  `json:"members,omitempty"` // Agent names
	Lead        string    `json:"lead,omitempty"`    // Team lead agent
}

// Store manages team configurations.
type Store struct {
	teamsDir string
}

// NewStore creates a new team store.
func NewStore(rootDir string) *Store {
	return &Store{
		teamsDir: filepath.Join(rootDir, ".bc", "teams"),
	}
}

// Init creates the teams directory if it doesn't exist.
func (s *Store) Init() error {
	return os.MkdirAll(s.teamsDir, 0750)
}

// Create creates a new team.
func (s *Store) Create(name string) (*Team, error) {
	if name == "" {
		return nil, fmt.Errorf("team name cannot be empty")
	}

	// Check if team already exists
	if s.Exists(name) {
		return nil, fmt.Errorf("team %q already exists", name)
	}

	now := time.Now().UTC()
	team := &Team{
		Name:      name,
		Members:   []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.save(team); err != nil {
		return nil, err
	}

	return team, nil
}

// Get retrieves a team by name.
func (s *Store) Get(name string) (*Team, error) {
	path := s.teamPath(name)
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from trusted teamsDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read team: %w", err)
	}

	var team Team
	if err := json.Unmarshal(data, &team); err != nil {
		return nil, fmt.Errorf("failed to parse team: %w", err)
	}

	return &team, nil
}

// List returns all teams.
func (s *Store) List() ([]*Team, error) {
	entries, err := os.ReadDir(s.teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read teams dir: %w", err)
	}

	var teams []*Team
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			team, err := s.Get(name)
			if err != nil {
				continue // Skip invalid entries
			}
			if team != nil {
				teams = append(teams, team)
			}
		}
	}

	return teams, nil
}

// Delete removes a team.
func (s *Store) Delete(name string) error {
	path := s.teamPath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("team %q not found", name)
		}
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// Exists checks if a team exists.
func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.teamPath(name))
	return err == nil
}

// Update modifies an existing team using the provided update function.
func (s *Store) Update(name string, updateFn func(*Team)) error {
	team, err := s.Get(name)
	if err != nil {
		return err
	}
	if team == nil {
		return fmt.Errorf("team %q not found", name)
	}

	updateFn(team)
	team.UpdatedAt = time.Now().UTC()

	return s.save(team)
}

// AddMember adds an agent to a team.
func (s *Store) AddMember(teamName, agentName string) error {
	return s.Update(teamName, func(t *Team) {
		// Check if already a member
		if slices.Contains(t.Members, agentName) {
			return
		}
		t.Members = append(t.Members, agentName)
	})
}

// RemoveMember removes an agent from a team.
func (s *Store) RemoveMember(teamName, agentName string) error {
	return s.Update(teamName, func(t *Team) {
		filtered := make([]string, 0, len(t.Members))
		for _, m := range t.Members {
			if m != agentName {
				filtered = append(filtered, m)
			}
		}
		t.Members = filtered
	})
}

// SetLead sets the team lead.
func (s *Store) SetLead(teamName, agentName string) error {
	return s.Update(teamName, func(t *Team) {
		t.Lead = agentName
	})
}

// SetDescription sets the team description.
func (s *Store) SetDescription(teamName, description string) error {
	return s.Update(teamName, func(t *Team) {
		t.Description = description
	})
}

// RemoveMemberFromAllTeams removes an agent from all teams.
// This is used when an agent is deleted to clean up stale team memberships.
func (s *Store) RemoveMemberFromAllTeams(agentName string) error {
	teams, err := s.List()
	if err != nil {
		return err
	}

	for _, t := range teams {
		if slices.Contains(t.Members, agentName) || t.Lead == agentName {
			err := s.Update(t.Name, func(team *Team) {
				// Remove from members list
				filtered := make([]string, 0, len(team.Members))
				for _, m := range team.Members {
					if m != agentName {
						filtered = append(filtered, m)
					}
				}
				team.Members = filtered

				// Clear lead if this agent was lead
				if team.Lead == agentName {
					team.Lead = ""
				}
			})
			if err != nil {
				return fmt.Errorf("failed to remove %s from team %s: %w", agentName, t.Name, err)
			}
		}
	}

	return nil
}

func (s *Store) save(team *Team) error {
	if err := s.Init(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(team, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal team: %w", err)
	}

	path := s.teamPath(team.Name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write team: %w", err)
	}

	return nil
}

func (s *Store) teamPath(name string) string {
	return filepath.Join(s.teamsDir, name+".json")
}
