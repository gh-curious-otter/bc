package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initBenchRepo creates a git repo for benchmarks with a specified number of commits.
func initBenchRepo(b *testing.B, commits int) string {
	b.Helper()
	dir := b.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "bench@test.com"},
		{"config", "user.name", "Benchmark"},
	} {
		cmd := exec.Command("git", args...) //nolint:gosec,noctx
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("git %v failed: %v (%s)", args, err, out)
		}
	}

	// Create commits
	for i := 0; i < commits; i++ {
		f := filepath.Join(dir, "file.txt")
		if err := os.WriteFile(f, []byte("content"+string(rune('0'+i%10))), 0o600); err != nil {
			b.Fatal(err)
		}
		cmd := exec.Command("git", "add", ".") //nolint:noctx
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("git add failed: %v (%s)", err, out)
		}
		cmd = exec.Command("git", "commit", "-m", "commit") //nolint:noctx
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("git commit failed: %v (%s)", err, out)
		}
	}

	b.Setenv("BC_AGENT_WORKTREE", "")
	return dir
}

func BenchmarkIsWriteOp_Write(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isWriteOp("commit")
	}
}

func BenchmarkIsWriteOp_Read(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isWriteOp("status")
	}
}

func BenchmarkIsWriteOp_AllWrites(b *testing.B) {
	ops := []string{"add", "commit", "push", "checkout", "reset", "clean",
		"merge", "rebase", "stash", "rm", "mv", "init", "pull",
		"cherry-pick", "revert", "tag", "branch"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, op := range ops {
			isWriteOp(op)
		}
	}
}

func BenchmarkValidateWorktree_NoEnv(b *testing.B) {
	b.Setenv("BC_AGENT_WORKTREE", "")
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateWorktree(dir)
	}
}

func BenchmarkValidateWorktree_Valid(b *testing.B) {
	dir := b.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0o750); err != nil {
		b.Fatal(err)
	}
	b.Setenv("BC_AGENT_WORKTREE", dir)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateWorktree(subdir)
	}
}

func BenchmarkValidateWorktree_Invalid(b *testing.B) {
	dir1 := b.TempDir()
	dir2 := b.TempDir()
	b.Setenv("BC_AGENT_WORKTREE", dir1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateWorktree(dir2)
	}
}

func BenchmarkStatus_Clean(b *testing.B) {
	dir := initBenchRepo(b, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Status(dir)
	}
}

func BenchmarkStatus_Dirty(b *testing.B) {
	dir := initBenchRepo(b, 1)
	// Create untracked files
	for j := 0; j < 10; j++ {
		f := filepath.Join(dir, "untracked"+string(rune('0'+j))+".txt")
		if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Status(dir)
	}
}

func BenchmarkDiff_NoChanges(b *testing.B) {
	dir := initBenchRepo(b, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Diff(dir)
	}
}

func BenchmarkDiff_WithChanges(b *testing.B) {
	dir := initBenchRepo(b, 1)
	// Modify the tracked file
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("modified content\nwith multiple lines\nof changes\n"), 0o600); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Diff(dir)
	}
}

func BenchmarkLog_1Commit(b *testing.B) {
	dir := initBenchRepo(b, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Log(dir, "--oneline")
	}
}

func BenchmarkLog_10Commits(b *testing.B) {
	dir := initBenchRepo(b, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Log(dir, "--oneline")
	}
}

func BenchmarkLog_100Commits(b *testing.B) {
	dir := initBenchRepo(b, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Log(dir, "--oneline")
	}
}

func BenchmarkRun_ReadOp(b *testing.B) {
	dir := initBenchRepo(b, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Run(dir, "status", "--short")
	}
}

func BenchmarkRun_WriteOpValidation(b *testing.B) {
	dir := initBenchRepo(b, 1)
	b.Setenv("BC_AGENT_WORKTREE", dir)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will fail (nothing to commit) but tests the validation path
		_ = Commit(dir, "benchmark")
	}
}

func BenchmarkAdd_SingleFile(b *testing.B) {
	dir := initBenchRepo(b, 1)
	f := filepath.Join(dir, "bench.txt")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		_ = Add(dir, "bench.txt")
	}
}

func BenchmarkCheckoutBranch(b *testing.B) {
	dir := initBenchRepo(b, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		branchName := "bench-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		_ = CheckoutBranch(dir, branchName)
	}
}
