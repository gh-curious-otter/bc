package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/process"
)

// resetProcessFlags resets process command flags between tests
func resetProcessFlags() {
	processCommand = ""
	processPort = 0
	processWorkDir = ""
	processLogLines = 50
}

func TestProcessListNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("process", "list")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestProcessListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("process", "list")
	if err != nil {
		t.Fatalf("process list returned error: %v", err)
	}
	if !strings.Contains(stdout, "No processes") {
		t.Errorf("expected 'No processes', got: %s", stdout)
	}
}

func TestProcessStopNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "stop", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessLogsNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "logs", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessShowNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "show", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessAttachNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "attach", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessStartRequiresCmd(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProcessFlags()
	defer resetProcessFlags()

	_, _, err := executeIntegrationCmd("process", "start", "test-proc")
	if err == nil {
		t.Fatal("expected error for missing --cmd flag, got nil")
	}
	// The error message varies by Cobra version but should indicate missing flag
	if !strings.Contains(err.Error(), "required") && !strings.Contains(err.Error(), "cmd") {
		t.Errorf("expected flag requirement error, got: %v", err)
	}
}

func TestProcessFlagDefaults(t *testing.T) {
	// Check process list flag defaults
	linesFlag := processLogsCmd.Flags().Lookup("lines")
	if linesFlag == nil {
		t.Fatal("lines flag not found")
	}
	if linesFlag.DefValue != "50" {
		t.Errorf("lines default: got %q, want %q", linesFlag.DefValue, "50")
	}

	// Check process start flag defaults
	portFlag := processStartCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("port flag not found")
	}
	if portFlag.DefValue != "0" {
		t.Errorf("port default: got %q, want %q", portFlag.DefValue, "0")
	}
}

func TestProcessStartNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	resetProcessFlags()
	processCommand = "echo hello"
	defer resetProcessFlags()

	_, _, err = executeIntegrationCmd("process", "start", "test", "--cmd", "echo hello")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestStatusStr(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		running bool
	}{
		{name: "running process", want: "running", running: true},
		{name: "stopped process", want: "stopped", running: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusStr(tt.running)
			if got != tt.want {
				t.Errorf("statusStr(%v) = %q, want %q", tt.running, got, tt.want)
			}
		})
	}
}

// seedProcesses creates process records in the workspace registry.
// Note: Register() always sets Running=true, so stopped processes can't be seeded this way.
func seedProcesses(t *testing.T, wsDir string, procs []*process.Process) {
	t.Helper()

	// Ensure processes directory exists
	procDir := filepath.Join(wsDir, ".bc", "processes")
	if err := os.MkdirAll(procDir, 0750); err != nil {
		t.Fatalf("failed to create processes dir: %v", err)
	}

	reg := process.NewRegistry(wsDir)
	for _, proc := range procs {
		if err := reg.Register(proc); err != nil {
			t.Fatalf("failed to register process %s: %v", proc.Name, err)
		}
	}
}

// seedProcess is a helper for single process seeding.
func seedProcess(t *testing.T, wsDir string, proc *process.Process) {
	seedProcesses(t, wsDir, []*process.Process{proc})
}

func TestProcessShowWithData(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed a process record
	proc := &process.Process{
		Name:      "test-server",
		Command:   "python -m http.server 8080",
		PID:       12345,
		Port:      8080,
		Owner:     "engineer-01",
		WorkDir:   "/tmp/test",
		StartedAt: time.Now().Add(-1 * time.Hour),
		Running:   true,
	}
	seedProcess(t, wsDir, proc)

	stdout, _, err := executeIntegrationCmd("process", "show", "test-server")
	if err != nil {
		t.Fatalf("process show error: %v", err)
	}

	// Check expected output fields
	if !strings.Contains(stdout, "test-server") {
		t.Errorf("output should contain process name, got: %s", stdout)
	}
	if !strings.Contains(stdout, "python -m http.server") {
		t.Errorf("output should contain command, got: %s", stdout)
	}
	if !strings.Contains(stdout, "12345") {
		t.Errorf("output should contain PID, got: %s", stdout)
	}
	if !strings.Contains(stdout, "8080") {
		t.Errorf("output should contain port, got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("output should contain owner, got: %s", stdout)
	}
}

func TestProcessShowWithoutOptionalFields(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed a minimal process record (no port, owner, workdir)
	proc := &process.Process{
		Name:      "minimal-proc",
		Command:   "sleep 100",
		PID:       99999,
		StartedAt: time.Now(),
	}
	seedProcess(t, wsDir, proc)

	stdout, _, err := executeIntegrationCmd("process", "show", "minimal-proc")
	if err != nil {
		t.Fatalf("process show error: %v", err)
	}

	if !strings.Contains(stdout, "minimal-proc") {
		t.Errorf("output should contain process name, got: %s", stdout)
	}
	// Registry.Register() always sets Running=true
	if !strings.Contains(stdout, "running") {
		t.Errorf("output should show running status, got: %s", stdout)
	}
	// Optional fields should not appear
	if strings.Contains(stdout, "Port:") {
		t.Errorf("output should not contain Port when not set, got: %s", stdout)
	}
	if strings.Contains(stdout, "Owner:") {
		t.Errorf("output should not contain Owner when not set, got: %s", stdout)
	}
}

func TestProcessListWithData(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed multiple processes using single registry instance
	procs := []*process.Process{
		{
			Name:      "web-server",
			Command:   "node server.js",
			PID:       1001,
			Port:      3000,
			StartedAt: time.Now(),
		},
		{
			Name:      "worker",
			Command:   "python worker.py",
			PID:       1002,
			StartedAt: time.Now().Add(-2 * time.Hour),
		},
	}

	seedProcesses(t, wsDir, procs)

	stdout, _, err := executeIntegrationCmd("process", "list")
	if err != nil {
		t.Fatalf("process list error: %v", err)
	}

	if !strings.Contains(stdout, "web-server") {
		t.Errorf("output should contain web-server, got: %s", stdout)
	}
	if !strings.Contains(stdout, "worker") {
		t.Errorf("output should contain worker, got: %s", stdout)
	}
}

func TestProcessLogsEmpty(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create process logs directory (logs are in .bc/processes/logs/)
	logsDir := filepath.Join(wsDir, ".bc", "processes", "logs")
	if err := os.MkdirAll(logsDir, 0750); err != nil {
		t.Fatalf("failed to create logs dir: %v", err)
	}

	// Create empty log file
	logPath := filepath.Join(logsDir, "empty-proc.log")
	if err := os.WriteFile(logPath, []byte{}, 0600); err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	// Seed process
	proc := &process.Process{
		Name:      "empty-proc",
		Command:   "echo hello",
		PID:       5555,
		StartedAt: time.Now(),
		Running:   true,
	}
	seedProcess(t, wsDir, proc)

	stdout, _, err := executeIntegrationCmd("process", "logs", "empty-proc")
	if err != nil {
		t.Fatalf("process logs error: %v", err)
	}

	if !strings.Contains(stdout, "No logs") {
		t.Errorf("expected 'No logs' message for empty log, got: %s", stdout)
	}
}

func TestProcessLogsWithContent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create process logs directory
	logsDir := filepath.Join(wsDir, ".bc", "processes", "logs")
	if err := os.MkdirAll(logsDir, 0750); err != nil {
		t.Fatalf("failed to create logs dir: %v", err)
	}

	// Create log file with content
	logPath := filepath.Join(logsDir, "logging-proc.log")
	logContent := "Starting server...\nListening on port 8080\nConnection accepted\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0600); err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	// Seed process
	proc := &process.Process{
		Name:      "logging-proc",
		Command:   "node app.js",
		PID:       6666,
		StartedAt: time.Now(),
		Running:   true,
	}
	seedProcess(t, wsDir, proc)

	stdout, _, err := executeIntegrationCmd("process", "logs", "logging-proc")
	if err != nil {
		t.Fatalf("process logs error: %v", err)
	}

	if !strings.Contains(stdout, "Starting server") {
		t.Errorf("output should contain log content, got: %s", stdout)
	}
	if !strings.Contains(stdout, "port 8080") {
		t.Errorf("output should contain log content, got: %s", stdout)
	}
}

// seedStoppedProcess writes a stopped process directly to the registry file.
// This bypasses Register() which always sets Running=true.
func seedStoppedProcess(t *testing.T, wsDir, name string) {
	t.Helper()

	procDir := filepath.Join(wsDir, ".bc", "processes")
	if err := os.MkdirAll(procDir, 0750); err != nil {
		t.Fatalf("failed to create processes dir: %v", err)
	}

	processes := map[string]*process.Process{
		name: {
			Name:      name,
			Command:   "test-cmd",
			PID:       0,
			Running:   false,
			StartedAt: time.Now(),
		},
	}

	data, err := json.MarshalIndent(processes, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal processes: %v", err)
	}

	regPath := filepath.Join(procDir, "registry.json")
	if err := os.WriteFile(regPath, data, 0600); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}
}

func TestProcessStopNotRunning(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed a stopped process
	seedStoppedProcess(t, wsDir, "stopped-proc")

	_, _, err := executeIntegrationCmd("process", "stop", "stopped-proc")
	if err == nil {
		t.Fatal("expected error for stopped process, got nil")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}

func TestProcessRestartNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "restart", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessRestartNotRunning(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed a stopped process
	seedStoppedProcess(t, wsDir, "stopped-proc")

	_, _, err := executeIntegrationCmd("process", "restart", "stopped-proc")
	if err == nil {
		t.Fatal("expected error for stopped process, got nil")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}
