package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// fileLock provides cross-process file locking using flock(2).
// This is used to coordinate git worktree and state file operations
// across multiple concurrent bc processes.
type fileLock struct {
	f    *os.File
	path string
}

// newFileLock creates a new file lock at the given path.
// The lock file is created if it doesn't exist.
func newFileLock(path string) *fileLock {
	return &fileLock{path: path}
}

// Lock acquires an exclusive flock with a timeout.
// Returns an error if the lock cannot be acquired within the timeout.
func (fl *fileLock) Lock(timeout time.Duration) error {
	if err := os.MkdirAll(filepath.Dir(fl.path), 0750); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}

	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			fl.f = f
			return nil
		}

		if time.Now().After(deadline) {
			_ = f.Close()
			return fmt.Errorf("timeout acquiring lock %s after %v", fl.path, timeout)
		}

		time.Sleep(interval)
	}
}

// Unlock releases the flock and closes the file.
func (fl *fileLock) Unlock() {
	if fl.f == nil {
		return
	}
	if err := syscall.Flock(int(fl.f.Fd()), syscall.LOCK_UN); err != nil {
		log.Warn("failed to unlock", "path", fl.path, "error", err)
	}
	_ = fl.f.Close()
	fl.f = nil
}

// worktreeLockPath returns the path for the worktree file lock.
func worktreeLockPath(workspace string) string {
	return filepath.Join(workspace, ".bc", "worktree.lock")
}

// stateLockPath returns the path for the agent state file lock.
func stateLockPath(stateDir string) string {
	return filepath.Join(stateDir, "agents.lock")
}
