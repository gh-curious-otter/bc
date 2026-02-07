package cmd

import (
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/demon"
)

func TestDemonCreate(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	output, err := executeCmd("demon", "create", "test-demon", "--schedule", "0 * * * *", "--cmd", "echo hello")
	if err != nil {
		t.Fatalf("demon create failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Created demon") {
		t.Errorf("expected confirmation message, got: %s", output)
	}
	if !strings.Contains(output, "test-demon") {
		t.Errorf("output should contain demon name: %s", output)
	}

	// Verify demon was created
	store := demon.NewStore(wsDir)
	d, err := store.Get("test-demon")
	if err != nil {
		t.Fatalf("failed to get demon: %v", err)
	}
	if d == nil {
		t.Fatal("demon not found after creation")
	}
	if d.Schedule != "0 * * * *" {
		t.Errorf("unexpected schedule: %s", d.Schedule)
	}
	if d.Command != "echo hello" {
		t.Errorf("unexpected command: %s", d.Command)
	}
}

func TestDemonCreateInvalidCron(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "create", "bad-demon", "--schedule", "invalid", "--cmd", "echo hello")
	if err == nil {
		t.Error("expected error for invalid cron syntax")
	}
	if !strings.Contains(err.Error(), "invalid cron") {
		t.Errorf("error should mention invalid cron: %v", err)
	}
}

func TestDemonCreateDuplicate(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "create", "dup-demon", "--schedule", "0 * * * *", "--cmd", "echo first")
	if err != nil {
		t.Fatalf("first demon create failed: %v", err)
	}

	_, err = executeCmd("demon", "create", "dup-demon", "--schedule", "0 * * * *", "--cmd", "echo second")
	if err == nil {
		t.Error("expected error for duplicate demon")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention already exists: %v", err)
	}
}

func TestDemonList(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create some demons
	store := demon.NewStore(wsDir)
	_, _ = store.Create("demon1", "0 * * * *", "echo one")
	_, _ = store.Create("demon2", "*/5 * * * *", "echo two")

	output, err := executeCmd("demon", "list")
	if err != nil {
		t.Fatalf("demon list failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "demon1") {
		t.Errorf("output should contain demon1: %s", output)
	}
	if !strings.Contains(output, "demon2") {
		t.Errorf("output should contain demon2: %s", output)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("output should contain header: %s", output)
	}
}

func TestDemonListEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("demon", "list")
	if err != nil {
		t.Fatalf("demon list failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No demons configured") {
		t.Errorf("output should indicate no demons: %s", output)
	}
}

func TestDemonShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a demon
	store := demon.NewStore(wsDir)
	_, _ = store.Create("show-demon", "0 9 * * 1-5", "bc backup")

	output, err := executeCmd("demon", "show", "show-demon")
	if err != nil {
		t.Fatalf("demon show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "show-demon") {
		t.Errorf("output should contain demon name: %s", output)
	}
	if !strings.Contains(output, "0 9 * * 1-5") {
		t.Errorf("output should contain schedule: %s", output)
	}
	if !strings.Contains(output, "bc backup") {
		t.Errorf("output should contain command: %s", output)
	}
}

func TestDemonShowNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "show", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent demon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDemonDelete(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a demon
	store := demon.NewStore(wsDir)
	_, _ = store.Create("delete-demon", "0 * * * *", "echo hello")

	output, err := executeCmd("demon", "delete", "delete-demon")
	if err != nil {
		t.Fatalf("demon delete failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Deleted demon") {
		t.Errorf("output should confirm deletion: %s", output)
	}

	// Verify deletion
	if store.Exists("delete-demon") {
		t.Error("demon should not exist after deletion")
	}
}

func TestDemonDeleteNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "delete", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent demon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}
