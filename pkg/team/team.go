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

// RemoveAgentFromAllTeams removes an agent from all teams they belong to.
// This should be called when an agent is deleted to maintain data integrity.
func (s *Store) RemoveAgentFromAllTeams(agentName string) error {
	teams, err := s.List()
	if err != nil {
		return err
	}

	for _, team := range teams {
		// Check if agent is a member
		if slices.Contains(team.Members, agentName) {
			if err := s.RemoveMember(team.Name, agentName); err != nil {
				return fmt.Errorf("failed to remove %s from team %s: %w", agentName, team.Name, err)
			}
		}
		// Also clear lead if this agent was the lead
		if team.Lead == agentName {
			if err := s.SetLead(team.Name, ""); err != nil {
				return fmt.Errorf("failed to clear lead for team %s: %w", team.Name, err)
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

// OrphanedMember represents a member that references a non-existent agent.
type OrphanedMember struct {
	TeamName   string
	MemberName string
	IsLead     bool
}

// FindOrphanedMembers finds all team members that reference non-existent agents.
// The agentExists function is called to check if each agent exists.
func (s *Store) FindOrphanedMembers(agentExists func(name string) bool) ([]OrphanedMember, error) {
	teams, err := s.List()
	if err != nil {
		return nil, err
	}

	var orphans []OrphanedMember
	for _, t := range teams {
		// Check members
		for _, member := range t.Members {
			if !agentExists(member) {
				orphans = append(orphans, OrphanedMember{
					TeamName:   t.Name,
					MemberName: member,
					IsLead:     false,
				})
			}
		}
		// Check lead
		if t.Lead != "" && !agentExists(t.Lead) {
			orphans = append(orphans, OrphanedMember{
				TeamName:   t.Name,
				MemberName: t.Lead,
				IsLead:     true,
			})
		}
	}

	return orphans, nil
}

// CleanupOrphanedMembers removes all team members that reference non-existent agents.
// Returns the number of orphaned members removed.
func (s *Store) CleanupOrphanedMembers(agentExists func(name string) bool) (int, error) {
	orphans, err := s.FindOrphanedMembers(agentExists)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, orphan := range orphans {
		if orphan.IsLead {
			if err := s.SetLead(orphan.TeamName, ""); err != nil {
				return removed, fmt.Errorf("failed to clear orphaned lead %s from team %s: %w",
					orphan.MemberName, orphan.TeamName, err)
			}
		} else {
			if err := s.RemoveMember(orphan.TeamName, orphan.MemberName); err != nil {
				return removed, fmt.Errorf("failed to remove orphaned member %s from team %s: %w",
					orphan.MemberName, orphan.TeamName, err)
			}
		}
		removed++
	}

	return removed, nil
}
