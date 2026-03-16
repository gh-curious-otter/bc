package daemon

import (
	"os"
	"testing"
)

func TestWriteReadPID(t *testing.T) {
	dir := t.TempDir()

	if err := WritePID(dir, 12345); err != nil {
		t.Fatalf("write pid: %v", err)
	}

	pid, err := ReadPID(dir)
	if err != nil {
		t.Fatalf("read pid: %v", err)
	}
	if pid != 12345 {
		t.Fatalf("expected 12345, got %d", pid)
	}
}

func TestRemovePID(t *testing.T) {
	dir := t.TempDir()

	if err := WritePID(dir, 1); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if err := RemovePID(dir); err != nil {
		t.Fatalf("remove pid: %v", err)
	}

	_, err := ReadPID(dir)
	if err == nil {
		t.Fatal("expected error after removal")
	}
}

func TestIsRunningCurrentProcess(t *testing.T) {
	dir := t.TempDir()

	// Write our own PID — should be running.
	if err := WritePID(dir, os.Getpid()); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if !IsRunning(dir) {
		t.Fatal("expected current process to be detected as running")
	}
}

func TestIsRunningNoPIDFile(t *testing.T) {
	dir := t.TempDir()
	if IsRunning(dir) {
		t.Fatal("expected false when no PID file exists")
	}
}

func TestWriteReadInfo(t *testing.T) {
	dir := t.TempDir()

	if err := WriteInfo(dir, "127.0.0.1:9374"); err != nil {
		t.Fatalf("write info: %v", err)
	}

	info, err := ReadInfo(dir)
	if err != nil {
		t.Fatalf("read info: %v", err)
	}
	if info.Addr != "127.0.0.1:9374" {
		t.Fatalf("expected addr 127.0.0.1:9374, got %s", info.Addr)
	}
	if info.PID != os.Getpid() {
		t.Fatalf("expected PID %d, got %d", os.Getpid(), info.PID)
	}
	if info.StartedAt.IsZero() {
		t.Fatal("expected non-zero started_at")
	}
}

func TestPIDPath(t *testing.T) {
	got := PIDPath("/tmp/test")
	want := "/tmp/test/bcd.pid"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
