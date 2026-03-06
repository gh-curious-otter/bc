package agent

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileLock_BasicLockUnlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	fl := newFileLock(path)
	if err := fl.Lock(time.Second); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}
	fl.Unlock()

	// Lock file should exist after locking
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file should exist: %v", err)
	}
}

func TestFileLock_DoubleUnlockSafe(t *testing.T) {
	dir := t.TempDir()
	fl := newFileLock(filepath.Join(dir, "test.lock"))
	if err := fl.Lock(time.Second); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}
	fl.Unlock()
	fl.Unlock() // should not panic
}

func TestFileLock_MutualExclusion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	fl1 := newFileLock(path)
	if err := fl1.Lock(time.Second); err != nil {
		t.Fatalf("fl1 Lock() failed: %v", err)
	}

	// Second lock should timeout quickly
	fl2 := newFileLock(path)
	err := fl2.Lock(300 * time.Millisecond)
	if err == nil {
		fl2.Unlock()
		fl1.Unlock()
		t.Fatal("expected fl2 Lock() to fail while fl1 holds the lock")
	}

	fl1.Unlock()

	// Now fl2 should succeed
	if err := fl2.Lock(time.Second); err != nil {
		t.Fatalf("fl2 Lock() after fl1 unlock failed: %v", err)
	}
	fl2.Unlock()
}

func TestFileLock_ConcurrentGoroutines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fl := newFileLock(path)
			if err := fl.Lock(10 * time.Second); err != nil {
				t.Errorf("Lock() failed: %v", err)
				return
			}
			// Increment non-atomically inside the lock to detect races
			v := atomic.LoadInt64(&counter)
			atomic.StoreInt64(&counter, v+1)
			fl.Unlock()
		}()
	}
	wg.Wait()

	if atomic.LoadInt64(&counter) != 10 {
		t.Fatalf("expected counter=10, got %d", atomic.LoadInt64(&counter))
	}
}

func TestFileLock_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "test.lock")

	fl := newFileLock(path)
	if err := fl.Lock(time.Second); err != nil {
		t.Fatalf("Lock() failed with nested dir: %v", err)
	}
	fl.Unlock()
}

func TestWorktreeLockPath(t *testing.T) {
	got := worktreeLockPath("/workspace")
	want := filepath.Join("/workspace", ".bc", "worktree.lock")
	if got != want {
		t.Fatalf("worktreeLockPath = %q, want %q", got, want)
	}
}
