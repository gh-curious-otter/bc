package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.TUIDir != "tui" {
		t.Errorf("TUIDir = %q, want %q", cfg.TUIDir, "tui")
	}
	if cfg.EntryPoint != "src/index.tsx" {
		t.Errorf("EntryPoint = %q, want %q", cfg.EntryPoint, "src/index.tsx")
	}
}

func TestNewInkBridge_MissingTUIDir(t *testing.T) {
	cfg := BridgeConfig{
		TUIDir:        "/nonexistent/path",
		WorkspaceRoot: "",
	}

	_, err := NewInkBridge(cfg)
	if err == nil {
		t.Error("expected error for missing TUI directory")
	}
}

func TestNewInkBridge_MissingEntryPoint(t *testing.T) {
	// Create a temp directory without entry point
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui")
	if err := os.MkdirAll(tuiDir, 0750); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		EntryPoint:    "src/index.tsx",
		WorkspaceRoot: tmpDir,
	}

	_, err := NewInkBridge(cfg)
	if err == nil {
		t.Error("expected error for missing entry point")
	}
}

func TestNewInkBridge_ValidSetup(t *testing.T) {
	// Create a temp directory with entry point
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui")
	srcDir := filepath.Join(tuiDir, "src")
	if err := os.MkdirAll(srcDir, 0750); err != nil {
		t.Fatal(err)
	}

	entryPoint := filepath.Join(srcDir, "index.tsx")
	if err := os.WriteFile(entryPoint, []byte("// placeholder"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		EntryPoint:    "src/index.tsx",
		WorkspaceRoot: tmpDir,
	}

	bridge, err := NewInkBridge(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}

	// cmd is nil until Start() is called
	if bridge.tuiDir == "" {
		t.Error("expected non-empty tuiDir")
	}
	if bridge.entryPoint == "" {
		t.Error("expected non-empty entryPoint")
	}
}

func TestInkBridge_CloseWithoutStart(t *testing.T) {
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui", "src")
	if err := os.MkdirAll(tuiDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tuiDir, "index.tsx"), []byte("//"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		WorkspaceRoot: tmpDir,
	}

	bridge, err := NewInkBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Close should not error even if not started
	if err := bridge.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestInkBridge_DoubleClose(t *testing.T) {
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui", "src")
	if err := os.MkdirAll(tuiDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tuiDir, "index.tsx"), []byte("//"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		WorkspaceRoot: tmpDir,
	}

	bridge, err := NewInkBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// First close
	if err := bridge.Close(); err != nil {
		t.Errorf("first Close() error = %v", err)
	}

	// Second close should be safe
	if err := bridge.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestInkBridge_SendSpecWithoutStart(t *testing.T) {
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui", "src")
	if err := os.MkdirAll(tuiDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tuiDir, "index.tsx"), []byte("//"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		WorkspaceRoot: tmpDir,
	}

	bridge, err := NewInkBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bridge.Close() }()

	err = bridge.SendSpec([]byte(`{"test": true}`))
	if err == nil {
		t.Error("expected error when sending spec without start")
	}
}

func TestInkBridge_ReadEventWithoutStart(t *testing.T) {
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui", "src")
	if err := os.MkdirAll(tuiDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tuiDir, "index.tsx"), []byte("//"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		WorkspaceRoot: tmpDir,
	}

	bridge, err := NewInkBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bridge.Close() }()

	_, err = bridge.ReadEvent()
	if err == nil {
		t.Error("expected error when reading event without start")
	}
}

func TestInkBridge_IsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	tuiDir := filepath.Join(tmpDir, "tui", "src")
	if err := os.MkdirAll(tuiDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tuiDir, "index.tsx"), []byte("//"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := BridgeConfig{
		TUIDir:        "tui",
		WorkspaceRoot: tmpDir,
	}

	bridge, err := NewInkBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bridge.Close() }()

	// Should not be running before start
	if bridge.IsRunning() {
		t.Error("expected IsRunning() = false before Start()")
	}
}
