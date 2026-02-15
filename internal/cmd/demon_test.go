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

func TestDemonRun(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a demon with a simple command
	store := demon.NewStore(wsDir)
	_, _ = store.Create("run-demon", "0 * * * *", "echo 'hello world'")

	output, err := executeCmd("demon", "run", "run-demon")
	if err != nil {
		t.Fatalf("demon run failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Running demon") {
		t.Errorf("output should indicate running: %s", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("output should contain command output: %s", output)
	}
	if !strings.Contains(output, "Run recorded") {
		t.Errorf("output should confirm recording: %s", output)
	}

	// Verify run count was incremented
	d, _ := store.Get("run-demon")
	if d.RunCount != 1 {
		t.Errorf("expected run count 1, got %d", d.RunCount)
	}
}

func TestDemonRunNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "run", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent demon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDemonEnable(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a demon and disable it first
	store := demon.NewStore(wsDir)
	_, _ = store.Create("enable-demon", "0 * * * *", "echo hello")
	_ = store.Disable("enable-demon")

	// Verify it's disabled
	d, _ := store.Get("enable-demon")
	if d.Enabled {
		t.Fatal("demon should be disabled before test")
	}

	output, err := executeCmd("demon", "enable", "enable-demon")
	if err != nil {
		t.Fatalf("demon enable failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Enabled demon") {
		t.Errorf("output should confirm enabling: %s", output)
	}

	// Verify it's now enabled
	d, _ = store.Get("enable-demon")
	if !d.Enabled {
		t.Error("demon should be enabled after command")
	}
}

func TestDemonEnableNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "enable", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent demon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDemonDisable(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a demon (enabled by default)
	store := demon.NewStore(wsDir)
	_, _ = store.Create("disable-demon", "0 * * * *", "echo hello")

	// Verify it's enabled
	d, _ := store.Get("disable-demon")
	if !d.Enabled {
		t.Fatal("demon should be enabled before test")
	}

	output, err := executeCmd("demon", "disable", "disable-demon")
	if err != nil {
		t.Fatalf("demon disable failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Disabled demon") {
		t.Errorf("output should confirm disabling: %s", output)
	}

	// Verify it's now disabled
	d, _ = store.Get("disable-demon")
	if d.Enabled {
		t.Error("demon should be disabled after command")
	}
}

func TestDemonDisableNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "disable", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent demon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

// --- Demon Name Validation Tests ---

func TestDemonCreateInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	tests := []struct {
		name    string
		wantErr string
	}{
		{"123-starts-with-digit", "must start with a letter"},
		{"has spaces", "must start with a letter"},
		{"has@special", "must start with a letter"},
		{"has.dot", "must start with a letter"},
		{"", "cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCmd("demon", "create", tt.name, "--schedule", "0 * * * *", "--cmd", "echo hello")
			if err == nil {
				t.Errorf("expected error for name %q", tt.name)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDemonShowInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "show", "123-invalid")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

func TestDemonDeleteInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "delete", "123invalid")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

func TestDemonRunInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "run", "has spaces")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

func TestDemonEnableInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "enable", "@invalid")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

func TestDemonDisableInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "disable", "123bad")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

func TestDemonLogsInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "logs", "bad name")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

func TestDemonEditInvalidName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "edit", "123badname", "--schedule", "0 * * * *")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "must start with a letter") {
		t.Errorf("error should mention invalid format: %v", err)
	}
}

// --- Demon Create Empty Command Test ---

func TestDemonCreateEmptyCommand(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("demon", "create", "test-demon", "--schedule", "0 * * * *", "--cmd", "")
	if err == nil {
		t.Error("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "command cannot be empty") {
		t.Errorf("error should mention empty command: %v", err)
	}
}

// --- Demon Edit Schedule Validation Test ---

func TestDemonEditInvalidSchedule(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a valid demon first
	store := demon.NewStore(wsDir)
	_, _ = store.Create("edit-sched", "0 * * * *", "echo hello")

	// Try to edit with invalid schedule
	_, err := executeCmd("demon", "edit", "edit-sched", "--schedule", "invalid-cron")
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
	if !strings.Contains(err.Error(), "invalid cron") {
		t.Errorf("error should mention invalid cron: %v", err)
	}
}

func TestDemonCreateValidNames(t *testing.T) {
	setupTestWorkspace(t)

	validNames := []string{
		"my-demon",
		"my_demon",
		"MyDemon",
		"_private",
		"backup",
		"test123",
		"a",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			_, err := executeCmd("demon", "create", name, "--schedule", "0 * * * *", "--cmd", "echo hello")
			if err != nil {
				t.Errorf("valid name %q should succeed: %v", name, err)
			}
		})
	}
}
