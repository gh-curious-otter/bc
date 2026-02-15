package cmd

import (
	"strings"
	"testing"

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
	_, _ = executeCmd("agent", "create", "engineer-01", "--role", "null")

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

	// Create an agent first (required by validation)
	_, _ = executeCmd("agent", "create", "agent-01", "--role", "null")

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
