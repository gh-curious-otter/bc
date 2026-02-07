package process

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry("/tmp/test")
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	expected := filepath.Join("/tmp/test", ".bc", "processes")
	if reg.processesDir != expected {
		t.Errorf("processesDir = %q, want %q", reg.processesDir, expected)
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p := &Process{
		Name:    "test-proc",
		Command: "echo hello",
		PID:     1234,
		Owner:   "engineer-01",
		Port:    8080,
	}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got := reg.Get("test-proc")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != "test-proc" {
		t.Errorf("Name = %q, want %q", got.Name, "test-proc")
	}
	if got.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", got.Command, "echo hello")
	}
	if got.PID != 1234 {
		t.Errorf("PID = %d, want %d", got.PID, 1234)
	}
	if got.Owner != "engineer-01" {
		t.Errorf("Owner = %q, want %q", got.Owner, "engineer-01")
	}
	if got.Port != 8080 {
		t.Errorf("Port = %d, want %d", got.Port, 8080)
	}
	if !got.Running {
		t.Error("Running should be true")
	}
	if got.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p := &Process{Name: "dup-proc", Command: "echo one"}
	if err := reg.Register(p); err != nil {
		t.Fatalf("First Register failed: %v", err)
	}

	p2 := &Process{Name: "dup-proc", Command: "echo two"}
	if err := reg.Register(p2); err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestRegistryUnregister(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p := &Process{Name: "delete-proc", Command: "echo hello"}
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := reg.Unregister("delete-proc"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	if got := reg.Get("delete-proc"); got != nil {
		t.Error("Get should return nil after Unregister")
	}
}

func TestRegistryUnregisterNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := reg.Unregister("nonexistent"); err == nil {
		t.Error("Expected error for nonexistent process")
	}
}

func TestRegistryList(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_ = reg.Register(&Process{Name: "proc-b", Command: "b"})
	_ = reg.Register(&Process{Name: "proc-a", Command: "a"})
	_ = reg.Register(&Process{Name: "proc-c", Command: "c"})

	procs := reg.List()
	if len(procs) != 3 {
		t.Fatalf("List returned %d processes, want 3", len(procs))
	}

	// Should be sorted by name
	if procs[0].Name != "proc-a" {
		t.Errorf("procs[0].Name = %q, want %q", procs[0].Name, "proc-a")
	}
	if procs[1].Name != "proc-b" {
		t.Errorf("procs[1].Name = %q, want %q", procs[1].Name, "proc-b")
	}
	if procs[2].Name != "proc-c" {
		t.Errorf("procs[2].Name = %q, want %q", procs[2].Name, "proc-c")
	}
}

func TestRegistryListByOwner(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_ = reg.Register(&Process{Name: "proc1", Command: "a", Owner: "engineer-01"})
	_ = reg.Register(&Process{Name: "proc2", Command: "b", Owner: "engineer-02"})
	_ = reg.Register(&Process{Name: "proc3", Command: "c", Owner: "engineer-01"})

	procs := reg.ListByOwner("engineer-01")
	if len(procs) != 2 {
		t.Fatalf("ListByOwner returned %d processes, want 2", len(procs))
	}
}

func TestRegistryMarkStopped(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p := &Process{Name: "stop-proc", Command: "echo", PID: 1234}
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := reg.MarkStopped("stop-proc"); err != nil {
		t.Fatalf("MarkStopped failed: %v", err)
	}

	got := reg.Get("stop-proc")
	if got.Running {
		t.Error("Running should be false after MarkStopped")
	}
	if got.PID != 0 {
		t.Errorf("PID should be 0 after MarkStopped, got %d", got.PID)
	}
}

func TestRegistryUpdatePID(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p := &Process{Name: "pid-proc", Command: "echo"}
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := reg.UpdatePID("pid-proc", 5678); err != nil {
		t.Fatalf("UpdatePID failed: %v", err)
	}

	got := reg.Get("pid-proc")
	if got.PID != 5678 {
		t.Errorf("PID = %d, want 5678", got.PID)
	}
}

func TestRegistryIsPortInUse(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_ = reg.Register(&Process{Name: "web", Command: "server", Port: 8080})

	if !reg.IsPortInUse(8080) {
		t.Error("Port 8080 should be in use")
	}
	if reg.IsPortInUse(8081) {
		t.Error("Port 8081 should not be in use")
	}
}

func TestRegistryGetByPort(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_ = reg.Register(&Process{Name: "web", Command: "server", Port: 8080})

	got := reg.GetByPort(8080)
	if got == nil {
		t.Fatal("GetByPort returned nil")
	}
	if got.Name != "web" {
		t.Errorf("Name = %q, want %q", got.Name, "web")
	}

	if reg.GetByPort(8081) != nil {
		t.Error("GetByPort(8081) should return nil")
	}
}

func TestRegistryClear(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_ = reg.Register(&Process{Name: "proc1", Command: "a"})
	_ = reg.Register(&Process{Name: "proc2", Command: "b"})

	if err := reg.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	procs := reg.List()
	if len(procs) != 0 {
		t.Errorf("List returned %d processes after Clear, want 0", len(procs))
	}
}

func TestRegistryPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and populate registry
	reg1 := NewRegistry(tmpDir)
	if err := reg1.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	p := &Process{
		Name:      "persist-proc",
		Command:   "echo",
		PID:       1234,
		Port:      3000,
		Owner:     "test",
		StartedAt: now,
	}
	if err := reg1.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Create new registry and load from disk
	reg2 := NewRegistry(tmpDir)
	if err := reg2.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	got := reg2.Get("persist-proc")
	if got == nil {
		t.Fatal("Process not found after reload")
	}
	if got.Name != "persist-proc" {
		t.Errorf("Name = %q, want %q", got.Name, "persist-proc")
	}
	if got.PID != 1234 {
		t.Errorf("PID = %d, want 1234", got.PID)
	}
	if got.Port != 3000 {
		t.Errorf("Port = %d, want 3000", got.Port)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if got := reg.Get("nonexistent"); got != nil {
		t.Error("Get should return nil for nonexistent process")
	}
}

func TestProcessStruct(t *testing.T) {
	p := Process{
		Name:      "test",
		Command:   "echo hello",
		PID:       123,
		Port:      8080,
		Owner:     "owner",
		WorkDir:   "/tmp",
		Running:   true,
		StartedAt: time.Now(),
	}

	if p.Name != "test" {
		t.Errorf("Name = %q, want %q", p.Name, "test")
	}
	if p.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", p.Command, "echo hello")
	}
	if p.PID != 123 {
		t.Errorf("PID = %d, want 123", p.PID)
	}
	if p.Port != 8080 {
		t.Errorf("Port = %d, want 8080", p.Port)
	}
	if p.Owner != "owner" {
		t.Errorf("Owner = %q, want %q", p.Owner, "owner")
	}
	if p.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q, want %q", p.WorkDir, "/tmp")
	}
	if !p.Running {
		t.Error("Running should be true")
	}
	if p.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
}

func TestRegistryProcessesDir(t *testing.T) {
	reg := NewRegistry("/home/user/project")
	expected := filepath.Join("/home/user/project", ".bc", "processes")
	if got := reg.ProcessesDir(); got != expected {
		t.Errorf("ProcessesDir() = %q, want %q", got, expected)
	}
}

func TestRegistryLogPath(t *testing.T) {
	reg := NewRegistry("/home/user/project")
	expected := filepath.Join("/home/user/project", ".bc", "processes", "logs", "web.log")
	if got := reg.LogPath("web"); got != expected {
		t.Errorf("LogPath() = %q, want %q", got, expected)
	}
}

func TestRegistryCreateLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	f, err := reg.CreateLogFile("test-proc")
	if err != nil {
		t.Fatalf("CreateLogFile failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Write some content
	_, _ = f.WriteString("test log line\n")

	// Verify file exists
	logPath := reg.LogPath("test-proc")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file should exist")
	}
}

func TestRegistryReadLogs(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create and write to log file
	f, err := reg.CreateLogFile("test-proc")
	if err != nil {
		t.Fatalf("CreateLogFile failed: %v", err)
	}

	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	for _, line := range lines {
		_, _ = f.WriteString(line + "\n")
	}
	_ = f.Close()

	// Read all lines
	content, err := reg.ReadLogs("test-proc", 0)
	if err != nil {
		t.Fatalf("ReadLogs failed: %v", err)
	}
	if content == "" {
		t.Error("expected log content, got empty")
	}

	// Read last 2 lines
	content, err = reg.ReadLogs("test-proc", 2)
	if err != nil {
		t.Fatalf("ReadLogs failed: %v", err)
	}
	if !strings.Contains(content, "line4") || !strings.Contains(content, "line5") {
		t.Errorf("expected last 2 lines, got: %s", content)
	}
}

func TestRegistryReadLogsNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	content, err := reg.ReadLogs("nonexistent", 10)
	if err != nil {
		t.Fatalf("ReadLogs should not error for nonexistent: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content, got: %s", content)
	}
}

func TestRegistryTailLogs(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create log file with content
	f, err := reg.CreateLogFile("test-proc")
	if err != nil {
		t.Fatalf("CreateLogFile failed: %v", err)
	}
	_, _ = f.WriteString("log output\n")
	_ = f.Close()

	content, err := reg.TailLogs("test-proc")
	if err != nil {
		t.Fatalf("TailLogs failed: %v", err)
	}
	if !strings.Contains(content, "log output") {
		t.Errorf("expected log output, got: %s", content)
	}
}

func TestRegistrySetLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p := &Process{Name: "log-proc", Command: "echo"}
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	logPath := "/custom/path/to.log"
	if err := reg.SetLogFile("log-proc", logPath); err != nil {
		t.Fatalf("SetLogFile failed: %v", err)
	}

	got := reg.Get("log-proc")
	if got.LogFile != logPath {
		t.Errorf("LogFile = %q, want %q", got.LogFile, logPath)
	}
}

func TestRegistrySetLogFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(tmpDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := reg.SetLogFile("nonexistent", "/path/to.log"); err == nil {
		t.Error("expected error for nonexistent process")
	}
}
