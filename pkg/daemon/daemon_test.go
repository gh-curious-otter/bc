package daemon

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	daemons, err := mgr.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(daemons) != 0 {
		t.Errorf("expected 0 daemons, got %d", len(daemons))
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	d, err := mgr.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d != nil {
		t.Errorf("expected nil, got %+v", d)
	}
}

func TestRunValidation(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	ctx := context.Background()

	tests := []struct {
		name    string
		opts    RunOptions
		wantErr string
	}{
		{
			name:    "missing name",
			opts:    RunOptions{Runtime: RuntimeBash, Cmd: "echo hi"},
			wantErr: "name is required",
		},
		{
			name:    "missing runtime",
			opts:    RunOptions{Name: "test"},
			wantErr: "runtime must be",
		},
		{
			name:    "bash without cmd",
			opts:    RunOptions{Name: "test", Runtime: RuntimeBash},
			wantErr: "--cmd is required",
		},
		{
			name:    "docker without image",
			opts:    RunOptions{Name: "test", Runtime: RuntimeDocker},
			wantErr: "--image is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.Run(ctx, tt.opts)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestRemoveNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	err = mgr.Remove(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error removing nonexistent daemon")
	}
}

func TestContainerName(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	name := mgr.containerName("mydb")
	if name == "" {
		t.Error("container name should not be empty")
	}
	if !strings.HasPrefix(name, "bc-") {
		t.Errorf("container name %q should start with bc-", name)
	}
}

func TestReadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/env"

	content := "# comment\nKEY1=value1\nKEY2=value2\n\nKEY3=value with spaces\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	env, err := readEnvFile(path)
	if err != nil {
		t.Fatalf("readEnvFile: %v", err)
	}
	if len(env) != 3 {
		t.Errorf("expected 3 env vars, got %d: %v", len(env), env)
	}
}

func TestNullStr(t *testing.T) {
	if nullStr("") != nil {
		t.Error("nullStr(\"\") should return nil")
	}
	s := nullStr("hello")
	if s == nil || *s != "hello" {
		t.Error("nullStr(\"hello\") should return pointer to \"hello\"")
	}
}
