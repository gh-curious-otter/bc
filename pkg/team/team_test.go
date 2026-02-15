package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/test")
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	expected := filepath.Join("/tmp/test", ".bc", "teams")
	if store.teamsDir != expected {
		t.Errorf("teamsDir = %q, want %q", store.teamsDir, expected)
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	teamsDir := filepath.Join(tmpDir, ".bc", "teams")
	if _, statErr := os.Stat(teamsDir); os.IsNotExist(statErr) {
		t.Errorf("Teams directory not created: %s", teamsDir)
	}
}

func TestStoreCreate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	team, err := store.Create("engineering")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if team.Name != "engineering" {
		t.Errorf("Name = %q, want %q", team.Name, "engineering")
	}
	if team.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if len(team.Members) != 0 {
		t.Errorf("Members should be empty, got %v", team.Members)
	}
}

func TestStoreCreateEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestStoreCreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("engineering")
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	_, err = store.Create("engineering")
	if err == nil {
		t.Error("Expected error for duplicate team")
	}
}

func TestStoreGet(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-team")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Get("test-team")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != "test-team" {
		t.Errorf("Name = %q, want %q", got.Name, "test-team")
	}
}

func TestStoreGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	got, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("Expected nil for nonexistent team")
	}
}

func TestStoreList(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, _ = store.Create("team1")
	_, _ = store.Create("team2")
	_, _ = store.Create("team3")

	teams, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(teams) != 3 {
		t.Errorf("List returned %d teams, want 3", len(teams))
	}
}

func TestStoreListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	teams, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(teams) != 0 {
		t.Errorf("List returned %d teams, want 0", len(teams))
	}
}

func TestStoreDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("deletable")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !store.Exists("deletable") {
		t.Error("Team should exist before delete")
	}

	err = store.Delete("deletable")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists("deletable") {
		t.Error("Team should not exist after delete")
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent team")
	}
}

func TestStoreUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("updatable")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.Update("updatable", func(t *Team) {
		t.Description = "test description"
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := store.Get("updatable")
	if got.Description != "test description" {
		t.Errorf("Description = %q, want %q", got.Description, "test description")
	}
}

func TestStoreUpdateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Update("nonexistent", func(t *Team) {
		t.Description = "test"
	})
	if err == nil {
		t.Error("Expected error for nonexistent team")
	}
}

func TestStoreAddMember(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("engineering")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.AddMember("engineering", "engineer-01")
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	got, _ := store.Get("engineering")
	if len(got.Members) != 1 {
		t.Fatalf("Members len = %d, want 1", len(got.Members))
	}
	if got.Members[0] != "engineer-01" {
		t.Errorf("Members[0] = %q, want %q", got.Members[0], "engineer-01")
	}

	// Add same member again - should not duplicate
	err = store.AddMember("engineering", "engineer-01")
	if err != nil {
		t.Fatalf("Second AddMember failed: %v", err)
	}

	got, _ = store.Get("engineering")
	if len(got.Members) != 1 {
		t.Errorf("Members len = %d, want 1 (no duplicate)", len(got.Members))
	}
}

func TestStoreRemoveMember(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("engineering")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_ = store.AddMember("engineering", "engineer-01")
	_ = store.AddMember("engineering", "engineer-02")

	err = store.RemoveMember("engineering", "engineer-01")
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	got, _ := store.Get("engineering")
	if len(got.Members) != 1 {
		t.Fatalf("Members len = %d, want 1", len(got.Members))
	}
	if got.Members[0] != "engineer-02" {
		t.Errorf("Members[0] = %q, want %q", got.Members[0], "engineer-02")
	}
}

func TestStoreSetLead(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("engineering")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.SetLead("engineering", "tech-lead-01")
	if err != nil {
		t.Fatalf("SetLead failed: %v", err)
	}

	got, _ := store.Get("engineering")
	if got.Lead != "tech-lead-01" {
		t.Errorf("Lead = %q, want %q", got.Lead, "tech-lead-01")
	}
}

func TestStoreSetDescription(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("engineering")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.SetDescription("engineering", "The engineering team")
	if err != nil {
		t.Fatalf("SetDescription failed: %v", err)
	}

	got, _ := store.Get("engineering")
	if got.Description != "The engineering team" {
		t.Errorf("Description = %q, want %q", got.Description, "The engineering team")
	}
}

func TestTeamPath(t *testing.T) {
	store := NewStore("/tmp/test")
	expected := filepath.Join("/tmp/test", ".bc", "teams", "my-team.json")
	got := store.teamPath("my-team")
	if got != expected {
		t.Errorf("teamPath = %q, want %q", got, expected)
	}
}

func TestStoreRemoveAgentFromAllTeams(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create multiple teams
	_, _ = store.Create("team-a")
	_, _ = store.Create("team-b")
	_, _ = store.Create("team-c")

	// Add agent to multiple teams
	_ = store.AddMember("team-a", "eng-01")
	_ = store.AddMember("team-a", "eng-02")
	_ = store.AddMember("team-b", "eng-01")
	_ = store.AddMember("team-c", "eng-03")

	// Set eng-01 as lead of team-b
	_ = store.SetLead("team-b", "eng-01")

	// Remove eng-01 from all teams
	err := store.RemoveAgentFromAllTeams("eng-01")
	if err != nil {
		t.Fatalf("RemoveAgentFromAllTeams failed: %v", err)
	}

	// Verify team-a no longer has eng-01
	teamA, _ := store.Get("team-a")
	for _, m := range teamA.Members {
		if m == "eng-01" {
			t.Error("team-a should not contain eng-01")
		}
	}
	if len(teamA.Members) != 1 || teamA.Members[0] != "eng-02" {
		t.Errorf("team-a members = %v, want [eng-02]", teamA.Members)
	}

	// Verify team-b no longer has eng-01 and lead is cleared
	teamB, _ := store.Get("team-b")
	if len(teamB.Members) != 0 {
		t.Errorf("team-b members = %v, want []", teamB.Members)
	}
	if teamB.Lead != "" {
		t.Errorf("team-b lead = %q, want empty (cleared)", teamB.Lead)
	}

	// Verify team-c unchanged
	teamC, _ := store.Get("team-c")
	if len(teamC.Members) != 1 || teamC.Members[0] != "eng-03" {
		t.Errorf("team-c members = %v, want [eng-03]", teamC.Members)
	}
}

func TestStoreRemoveAgentFromAllTeams_NoTeams(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Should not error when no teams exist
	err := store.RemoveAgentFromAllTeams("eng-01")
	if err != nil {
		t.Fatalf("RemoveAgentFromAllTeams with no teams failed: %v", err)
	}
}

func TestStoreRemoveAgentFromAllTeams_AgentNotInAnyTeam(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, _ = store.Create("team-a")
	_ = store.AddMember("team-a", "eng-02")

	// Remove agent that doesn't exist in any team
	err := store.RemoveAgentFromAllTeams("eng-01")
	if err != nil {
		t.Fatalf("RemoveAgentFromAllTeams failed: %v", err)
	}

	// Verify team unchanged
	team, _ := store.Get("team-a")
	if len(team.Members) != 1 || team.Members[0] != "eng-02" {
		t.Errorf("team-a members = %v, want [eng-02]", team.Members)
	}
}
