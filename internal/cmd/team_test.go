package cmd

import (
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/team"
)

func TestTeamCreate(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	output, err := executeCmd("team", "create", "engineering")
	if err != nil {
		t.Fatalf("team create failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Created team") {
		t.Errorf("expected confirmation message, got: %s", output)
	}
	if !strings.Contains(output, "engineering") {
		t.Errorf("output should contain team name: %s", output)
	}

	// Verify team was created
	store := team.NewStore(wsDir)
	tm, getErr := store.Get("engineering")
	if getErr != nil {
		t.Fatalf("failed to get team: %v", getErr)
	}
	if tm == nil {
		t.Fatal("team not found after creation")
	}
}

func TestTeamCreateDuplicate(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "create", "dup-team")
	if err != nil {
		t.Fatalf("first team create failed: %v", err)
	}

	_, err = executeCmd("team", "create", "dup-team")
	if err == nil {
		t.Error("expected error for duplicate team")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention already exists: %v", err)
	}
}

func TestTeamList(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create some teams
	store := team.NewStore(wsDir)
	_, _ = store.Create("team1")
	_, _ = store.Create("team2")

	output, err := executeCmd("team", "list")
	if err != nil {
		t.Fatalf("team list failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "team1") {
		t.Errorf("output should contain team1: %s", output)
	}
	if !strings.Contains(output, "team2") {
		t.Errorf("output should contain team2: %s", output)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("output should contain header: %s", output)
	}
}

func TestTeamListEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("team", "list")
	if err != nil {
		t.Fatalf("team list failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No teams configured") {
		t.Errorf("output should indicate no teams: %s", output)
	}
}

func TestTeamShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a team with members
	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")
	_ = store.SetDescription("engineering", "The engineering team")
	_ = store.SetLead("engineering", "tech-lead-01")
	_ = store.AddMember("engineering", "engineer-01")
	_ = store.AddMember("engineering", "engineer-02")

	output, err := executeCmd("team", "show", "engineering")
	if err != nil {
		t.Fatalf("team show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "engineering") {
		t.Errorf("output should contain team name: %s", output)
	}
	if !strings.Contains(output, "The engineering team") {
		t.Errorf("output should contain description: %s", output)
	}
	if !strings.Contains(output, "tech-lead-01") {
		t.Errorf("output should contain lead: %s", output)
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("output should contain member: %s", output)
	}
}

func TestTeamShowNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "show", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestTeamDelete(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a team
	store := team.NewStore(wsDir)
	_, _ = store.Create("deletable")

	output, err := executeCmd("team", "delete", "deletable")
	if err != nil {
		t.Fatalf("team delete failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Deleted team") {
		t.Errorf("output should confirm deletion: %s", output)
	}

	// Verify deletion
	if store.Exists("deletable") {
		t.Error("team should not exist after deletion")
	}
}

func TestTeamDeleteNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "delete", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestTeamAdd(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a team
	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")

	// Create an agent first (required by validation)
	// Use seedAgents to properly create the agent in the state
	agents := map[string]*agent.Agent{
		"engineer-01": {
			Name:      "engineer-01",
			Role:      agent.Role("engineer"),
			State:     agent.StateIdle,
			Tool:      "claude",
			Session:   "test-session",
			Workspace: wsDir,
			ID:        "eng-01-id",
		},
	}
	seedAgents(t, wsDir, agents)

	output, err := executeCmd("team", "add", "engineering", "engineer-01")
	if err != nil {
		t.Fatalf("team add failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Added") {
		t.Errorf("output should confirm addition: %s", output)
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("output should contain agent name: %s", output)
	}

	// Verify member was added
	tm, _ := store.Get("engineering")
	if len(tm.Members) != 1 {
		t.Fatalf("Members len = %d, want 1", len(tm.Members))
	}
	if tm.Members[0] != "engineer-01" {
		t.Errorf("Members[0] = %q, want %q", tm.Members[0], "engineer-01")
	}
}

func TestTeamAddTeamNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "add", "nonexistent", "agent-01")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestTeamRemove(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a team with a member
	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")
	_ = store.AddMember("engineering", "engineer-01")
	_ = store.AddMember("engineering", "engineer-02")

	output, err := executeCmd("team", "remove", "engineering", "engineer-01")
	if err != nil {
		t.Fatalf("team remove failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Removed") {
		t.Errorf("output should confirm removal: %s", output)
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("output should contain agent name: %s", output)
	}

	// Verify member was removed
	tm, _ := store.Get("engineering")
	if len(tm.Members) != 1 {
		t.Fatalf("Members len = %d, want 1", len(tm.Members))
	}
	if tm.Members[0] != "engineer-02" {
		t.Errorf("Members[0] = %q, want %q", tm.Members[0], "engineer-02")
	}
}

func TestTeamRemoveTeamNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "remove", "nonexistent", "agent-01")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestTeamAddAgentNotFound(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a team but don't create the agent
	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")

	_, err := executeCmd("team", "add", "engineering", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention agent does not exist: %v", err)
	}
}

// --- Team Rename Tests ---

func TestTeamRename_Basic(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("old-name")

	output, err := executeCmd("team", "rename", "old-name", "new-name")
	if err != nil {
		t.Fatalf("rename failed: %v", err)
	}

	if !strings.Contains(output, "new-name") {
		t.Errorf("output should contain new name: %s", output)
	}

	// Verify old team is gone
	if store.Exists("old-name") {
		t.Error("old team should not exist after rename")
	}
	// Verify new team exists
	if !store.Exists("new-name") {
		t.Error("new team should exist after rename")
	}
}

func TestTeamRename_NonExistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "rename", "nonexistent", "new-name")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
}

func TestTeamRename_ToExisting(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("team1")
	_, _ = store.Create("team2")

	_, err := executeCmd("team", "rename", "team1", "team2")
	if err == nil {
		t.Error("expected error when renaming to existing team name")
	}
}

func TestTeamRename_PreservesMembers(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("team-to-rename")
	_ = store.AddMember("team-to-rename", "agent1")
	_ = store.AddMember("team-to-rename", "agent2")

	_, err := executeCmd("team", "rename", "team-to-rename", "renamed-team")
	if err != nil {
		t.Fatalf("rename failed: %v", err)
	}

	// Verify members are preserved
	newTeam, _ := store.Get("renamed-team")
	if len(newTeam.Members) != 2 {
		t.Errorf("expected 2 members after rename, got %d", len(newTeam.Members))
	}
}

// --- Team Remove Tests ---

func TestTeamRemove_NonExistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "remove", "nonexistent-team", "agent-01")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestTeamRemove_AllMembers(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("team-empty")
	_ = store.AddMember("team-empty", "agent1")
	_ = store.AddMember("team-empty", "agent2")

	// Remove both members
	_, err := executeCmd("team", "remove", "team-empty", "agent1")
	if err != nil {
		t.Fatalf("remove first member failed: %v", err)
	}

	_, err = executeCmd("team", "remove", "team-empty", "agent2")
	if err != nil {
		t.Fatalf("remove second member failed: %v", err)
	}

	// Verify team is empty
	tm, _ := store.Get("team-empty")
	if len(tm.Members) != 0 {
		t.Errorf("team should be empty, got %d members", len(tm.Members))
	}
}

// --- Team Show Extended Tests ---

func TestTeamShow_WithMembers(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")
	_ = store.AddMember("engineering", "eng-01")
	_ = store.AddMember("engineering", "eng-02")
	_ = store.SetLead("engineering", "lead-01")
	_ = store.SetDescription("engineering", "Backend team")

	_, err := executeCmd("team", "show", "engineering")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
}

func TestTeamShow_NonExistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "show", "nonexistent-team")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

// --- Team List Extended Tests ---

func TestTeamList_Multiple(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")
	_, _ = store.Create("product")
	_, _ = store.Create("design")

	_, err := executeCmd("team", "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestTeamList_WithMembers(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("engineering")
	_ = store.AddMember("engineering", "eng-01")
	_ = store.AddMember("engineering", "eng-02")

	_, err := executeCmd("team", "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestTeamList_Empty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("team", "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	// Should mention no teams
	if !strings.Contains(output, "No teams") {
		t.Errorf("should mention no teams: %s", output)
	}
}

// --- Team Create Extended Tests ---

func TestTeamCreate_MultipleTeams(t *testing.T) {
	setupTestWorkspace(t)

	teams := []string{"engineering", "product", "design", "marketing"}
	for _, teamName := range teams {
		_, err := executeCmd("team", "create", teamName)
		if err != nil {
			t.Errorf("create team %s failed: %v", teamName, err)
		}
	}
}

func TestTeamCreate_DuplicateCheck(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "create", "duplicate-team")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = executeCmd("team", "create", "duplicate-team")
	if err == nil {
		t.Error("expected error for duplicate team")
	}
	if !strings.Contains(err.Error(), "exists") {
		t.Errorf("error should mention exists: %v", err)
	}
}

// --- Team Delete Extended Tests ---

func TestTeamDelete_NonExistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("team", "delete", "nonexistent-team")
	if err == nil {
		t.Error("expected error for nonexistent team")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestTeamDelete_WithMembers(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := team.NewStore(wsDir)
	_, _ = store.Create("delete-me")
	_ = store.AddMember("delete-me", "agent1")

	_, err := executeCmd("team", "delete", "delete-me")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify team is gone
	if store.Exists("delete-me") {
		t.Error("team should not exist after deletion")
	}
}
