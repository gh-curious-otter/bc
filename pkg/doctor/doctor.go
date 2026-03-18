// Package doctor provides workspace health checks and diagnostics for bc.
//
// Run a full health check:
//
//	report := doctor.RunAll(ctx, ws)
//	for _, cat := range report.Categories {
//	    fmt.Println(cat.Name)
//	    for _, item := range cat.Items {
//	        fmt.Printf("  %s %s\n", item.Status, item.Message)
//	    }
//	}
//
// Run a single category:
//
//	cat := doctor.CheckWorkspace(ws)
//	cat := doctor.CheckDatabase(ws)
//	cat := doctor.CheckAgents(ctx, ws)
//	cat := doctor.CheckTools(ctx)
//	cat := doctor.CheckGit(ctx, ws)
package doctor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/provider"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Severity indicates the outcome of a single health check item.
type Severity int

const (
	// SeverityOK means the check passed.
	SeverityOK Severity = iota
	// SeverityWarn means a non-critical issue was found.
	SeverityWarn
	// SeverityFail means a critical issue was found.
	SeverityFail
)

// String returns the string representation of a Severity.
func (s Severity) String() string {
	switch s {
	case SeverityOK:
		return "ok"
	case SeverityWarn:
		return "warn"
	default:
		return "fail"
	}
}

// Item is the result of a single health check.
// Field order optimised by fieldalignment.
type Item struct {
	Name     string
	Message  string
	Fix      string
	Severity Severity
}

// CategoryReport is the result of checking one category.
type CategoryReport struct {
	Name  string
	Items []Item
}

// Counts tallies ok/warn/fail across Items.
func (c *CategoryReport) Counts() (ok, warn, fail int) {
	for _, it := range c.Items {
		switch it.Severity {
		case SeverityOK:
			ok++
		case SeverityWarn:
			warn++
		case SeverityFail:
			fail++
		}
	}
	return
}

// Report contains all category results from a full health check.
type Report struct {
	Categories []CategoryReport
}

// Summary returns aggregate ok/warn/fail totals across all categories.
func (r *Report) Summary() (ok, warn, fail int) {
	for i := range r.Categories {
		o, w, f := r.Categories[i].Counts()
		ok += o
		warn += w
		fail += f
	}
	return
}

// RunAll runs all health check categories and returns a combined report.
func RunAll(ctx context.Context, ws *workspace.Workspace) *Report {
	cats := []CategoryReport{
		CheckWorkspace(ws),
		CheckDatabase(ws),
		CheckAgents(ctx, ws),
		CheckTools(ctx),
		CheckGit(ctx, ws),
	}
	return &Report{Categories: cats}
}

// CategoryByName runs a single named category check.
// Returns nil if the category name is unknown.
func CategoryByName(ctx context.Context, ws *workspace.Workspace, name string) *CategoryReport {
	switch strings.ToLower(name) {
	case "workspace":
		c := CheckWorkspace(ws)
		return &c
	case "database", "db":
		c := CheckDatabase(ws)
		return &c
	case "agents", "agent":
		c := CheckAgents(ctx, ws)
		return &c
	case "tools", "tool":
		c := CheckTools(ctx)
		return &c
	case "git":
		c := CheckGit(ctx, ws)
		return &c
	default:
		return nil
	}
}

// ValidCategories returns the list of valid category names.
func ValidCategories() []string {
	return []string{"workspace", "database", "agents", "tools", "git"}
}

// ─── Workspace ───────────────────────────────────────────────────────────────

// CheckWorkspace checks the .bc/ directory structure, config validity, and roles.
func CheckWorkspace(ws *workspace.Workspace) CategoryReport {
	cat := CategoryReport{Name: "Workspace"}

	stateDir := ws.StateDir()

	// .bc/ directory
	if _, err := os.Stat(stateDir); err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     ".bc/ directory",
			Message:  "missing",
			Severity: SeverityFail,
			Fix:      "run 'bc init' to initialize the workspace",
		})
		return cat
	}
	cat.Items = append(cat.Items, Item{
		Name:     ".bc/ directory",
		Message:  "exists",
		Severity: SeverityOK,
	})

	// config.toml
	configPath := filepath.Join(stateDir, "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     "config.toml",
			Message:  "missing",
			Severity: SeverityFail,
			Fix:      "run 'bc init' to initialize the workspace",
		})
	} else {
		if ws.Config != nil {
			if err := ws.Config.Validate(); err != nil {
				cat.Items = append(cat.Items, Item{
					Name:     "config.toml",
					Message:  fmt.Sprintf("invalid: %v", err),
					Severity: SeverityFail,
					Fix:      "edit .bc/config.toml to correct the error",
				})
			} else {
				cat.Items = append(cat.Items, Item{
					Name:     "config.toml",
					Message:  fmt.Sprintf("valid (workspace: %s)", ws.Config.Workspace.Name),
					Severity: SeverityOK,
				})
			}
		} else {
			cat.Items = append(cat.Items, Item{
				Name:     "config.toml",
				Message:  "present",
				Severity: SeverityOK,
			})
		}
	}

	// Roles directory
	rolesDir := ws.RolesDir()
	if _, err := os.Stat(rolesDir); err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     "roles/",
			Message:  "missing",
			Severity: SeverityWarn,
			Fix:      "run 'bc init' to recreate role files",
		})
	} else {
		entries, err := os.ReadDir(rolesDir)
		if err != nil {
			cat.Items = append(cat.Items, Item{
				Name:     "roles/",
				Message:  fmt.Sprintf("unreadable: %v", err),
				Severity: SeverityWarn,
			})
		} else {
			count := 0
			for _, e := range entries {
				if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
					count++
				}
			}
			msg := fmt.Sprintf("%d role file(s) defined", count)
			sev := SeverityOK
			if count == 0 {
				sev = SeverityWarn
				msg = "no role files found"
			}
			cat.Items = append(cat.Items, Item{
				Name:     "roles/",
				Message:  msg,
				Severity: sev,
			})
		}
	}

	// agents/ directory
	agentsDir := ws.AgentsDir()
	if _, err := os.Stat(agentsDir); err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     "agents/",
			Message:  "missing",
			Severity: SeverityWarn,
			Fix:      "run 'bc init' to recreate directory structure",
		})
	} else {
		cat.Items = append(cat.Items, Item{
			Name:     "agents/",
			Message:  "exists",
			Severity: SeverityOK,
		})
	}

	return cat
}

// ─── Database ────────────────────────────────────────────────────────────────

// CheckDatabase checks SQLite integrity and table existence.
func CheckDatabase(ws *workspace.Workspace) CategoryReport {
	cat := CategoryReport{Name: "Database"}

	stateDir := ws.StateDir()

	// state.db
	stateDB := filepath.Join(stateDir, "state.db")
	cat.Items = append(cat.Items, checkSQLiteFile(stateDB, "state.db", []string{"agents"})...)

	// channels.db
	channelsDB := filepath.Join(stateDir, "channels.db")
	cat.Items = append(cat.Items, checkSQLiteFile(channelsDB, "channels.db", []string{"channels", "messages"})...)

	return cat
}

// checkSQLiteFile checks a SQLite file: existence, integrity, and required tables.
func checkSQLiteFile(path, label string, requiredTables []string) []Item {
	// 1 for file check, 1 for integrity, len(requiredTables) for table checks
	items := make([]Item, 0, 2+len(requiredTables))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		items = append(items, Item{
			Name:     label,
			Message:  "not found (will be created on first use)",
			Severity: SeverityWarn,
		})
		return items
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&mode=ro")
	if err != nil {
		items = append(items, Item{
			Name:     label,
			Message:  fmt.Sprintf("cannot open: %v", err),
			Severity: SeverityFail,
			Fix:      fmt.Sprintf("check file permissions on %s", path),
		})
		return items
	}
	defer func() { _ = db.Close() }()

	// PRAGMA integrity_check
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		items = append(items, Item{
			Name:     label + " integrity",
			Message:  fmt.Sprintf("check failed: %v", err),
			Severity: SeverityFail,
		})
	} else if result == "ok" {
		items = append(items, Item{
			Name:     label + " integrity",
			Message:  "ok",
			Severity: SeverityOK,
		})
	} else {
		items = append(items, Item{
			Name:     label + " integrity",
			Message:  result,
			Severity: SeverityFail,
		})
	}

	// Check required tables
	for _, table := range requiredTables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err == sql.ErrNoRows {
			items = append(items, Item{
				Name:     fmt.Sprintf("%s: table %q", label, table),
				Message:  "missing",
				Severity: SeverityFail,
				Fix:      "run 'bc doctor fix' to recreate missing tables",
			})
		} else if err != nil {
			items = append(items, Item{
				Name:     fmt.Sprintf("%s: table %q", label, table),
				Message:  fmt.Sprintf("query failed: %v", err),
				Severity: SeverityFail,
			})
		} else {
			items = append(items, Item{
				Name:     fmt.Sprintf("%s: table %q", label, table),
				Message:  "present",
				Severity: SeverityOK,
			})
		}
	}

	return items
}

// ─── Agents ──────────────────────────────────────────────────────────────────

// staleAgentThreshold is how long without an update before flagging an agent as potentially stuck.
const staleAgentThreshold = 2 * time.Hour

// CheckAgents checks for orphaned sessions and stale agents.
func CheckAgents(ctx context.Context, ws *workspace.Workspace) CategoryReport {
	cat := CategoryReport{Name: "Agents"}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err := mgr.LoadState(); err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     "agent state",
			Message:  fmt.Sprintf("failed to load: %v", err),
			Severity: SeverityWarn,
		})
		return cat
	}

	agents := mgr.ListAgents()
	if len(agents) == 0 {
		cat.Items = append(cat.Items, Item{
			Name:     "agents",
			Message:  "no agents defined",
			Severity: SeverityOK,
		})
		return cat
	}

	healthy := 0
	for _, a := range agents {
		// Skip stopped/done agents
		if a.State == agent.StateStopped || a.State == agent.StateDone {
			continue
		}

		agentOK := true

		// Check worktree directory exists (if set)
		if a.WorktreeDir != "" {
			if _, err := os.Stat(a.WorktreeDir); err != nil {
				cat.Items = append(cat.Items, Item{
					Name:     a.Name,
					Message:  fmt.Sprintf("worktree missing: %s", a.WorktreeDir),
					Severity: SeverityFail,
					Fix:      "run 'bc doctor fix' to remove orphaned agent entries",
				})
				agentOK = false
			}
		}

		// Check for stale state (active but no recent update)
		if agentOK && (a.State == agent.StateWorking || a.State == agent.StateIdle) {
			if time.Since(a.UpdatedAt) > staleAgentThreshold {
				cat.Items = append(cat.Items, Item{
					Name:     a.Name,
					Message:  fmt.Sprintf("no activity for %s (may be stuck)", formatDuration(time.Since(a.UpdatedAt))),
					Severity: SeverityWarn,
				})
				agentOK = false
			}
		}

		if agentOK {
			healthy++
		}
	}

	if healthy > 0 {
		cat.Items = append(cat.Items, Item{
			Name:     "agents",
			Message:  fmt.Sprintf("%d agent(s) healthy", healthy),
			Severity: SeverityOK,
		})
	}

	return cat
}

// ─── Tools ───────────────────────────────────────────────────────────────────

// CheckTools checks binary installations: tmux, git, and registered providers.
func CheckTools(ctx context.Context) CategoryReport {
	cat := CategoryReport{Name: "Tools"}

	// Required tools
	required := []struct {
		name string
		fix  string
	}{
		{"tmux", "brew install tmux  OR  apt install tmux"},
		{"git", "brew install git   OR  apt install git"},
	}
	for _, t := range required {
		cat.Items = append(cat.Items, checkBinary(ctx, t.name, t.fix))
	}

	// Optional: registered AI providers
	for _, p := range provider.ListProviders() {
		item := Item{Name: p.Name(), Fix: p.InstallHint()}
		if !p.IsInstalled(ctx) {
			item.Message = "not found"
			item.Severity = SeverityWarn
		} else {
			version := p.Version(ctx)
			path, _ := exec.LookPath(p.Binary())
			if version != "" && path != "" {
				item.Message = fmt.Sprintf("%s (%s)", path, version)
			} else if path != "" {
				item.Message = path
			} else {
				item.Message = "installed"
			}
			item.Severity = SeverityOK
		}
		cat.Items = append(cat.Items, item)
	}

	return cat
}

// checkBinary checks whether a binary is in PATH.
func checkBinary(ctx context.Context, name, fix string) Item {
	path, err := exec.LookPath(name)
	if err != nil {
		return Item{
			Name:     name,
			Message:  "not found",
			Severity: SeverityFail,
			Fix:      fix,
		}
	}

	version := binaryVersion(ctx, name)
	msg := path
	if version != "" {
		msg = fmt.Sprintf("%s (%s)", path, version)
	}
	return Item{Name: name, Message: msg, Severity: SeverityOK}
}

// binaryVersion tries to get a version string for common binaries.
func binaryVersion(ctx context.Context, name string) string {
	var args []string
	switch name {
	case "tmux":
		args = []string{"-V"}
	case "git":
		args = []string{"--version"}
	default:
		return ""
	}

	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}

// ─── Git ─────────────────────────────────────────────────────────────────────

// CheckGit checks git worktree health for the workspace.
func CheckGit(ctx context.Context, ws *workspace.Workspace) CategoryReport {
	cat := CategoryReport{Name: "Git"}

	// Verify git is available
	if _, err := exec.LookPath("git"); err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     "git",
			Message:  "not found — cannot check worktrees",
			Severity: SeverityFail,
			Fix:      "install git",
		})
		return cat
	}

	// List worktrees via git
	cmd := exec.CommandContext(ctx, "git", "-C", ws.RootDir, "worktree", "list", "--porcelain") //nolint:gosec // G204: args are derived from workspace config, not user input
	out, err := cmd.Output()
	if err != nil {
		cat.Items = append(cat.Items, Item{
			Name:     "git worktrees",
			Message:  fmt.Sprintf("could not list: %v", err),
			Severity: SeverityWarn,
		})
		return cat
	}

	valid, orphaned := parseWorktrees(string(out), ws.RootDir)

	cat.Items = append(cat.Items, Item{
		Name:     "git worktrees",
		Message:  fmt.Sprintf("%d valid", valid),
		Severity: SeverityOK,
	})

	for _, path := range orphaned {
		cat.Items = append(cat.Items, Item{
			Name:     "orphaned worktree",
			Message:  path,
			Severity: SeverityWarn,
			Fix:      fmt.Sprintf("git worktree remove --force %q", path),
		})
	}

	return cat
}

// parseWorktrees parses `git worktree list --porcelain` output.
// Returns (valid count, list of orphaned paths).
// A worktree is orphaned if its directory no longer exists and it is not the main worktree.
func parseWorktrees(output, rootDir string) (valid int, orphaned []string) {
	blocks := strings.Split(strings.TrimSpace(output), "\n\n")
	for i, block := range blocks {
		var wt string
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "worktree ") {
				wt = strings.TrimPrefix(line, "worktree ")
				break
			}
		}
		if wt == "" {
			continue
		}
		// Skip the main worktree (first entry)
		if i == 0 || wt == rootDir {
			valid++
			continue
		}
		if _, err := os.Stat(wt); err != nil {
			orphaned = append(orphaned, wt)
		} else {
			valid++
		}
	}
	return
}

// ─── Fix ─────────────────────────────────────────────────────────────────────

// FixResult describes one auto-fix action taken (or that would be taken).
type FixResult struct {
	Action  string
	Success bool
	Message string
}

// Fix runs auto-fix actions for all fixable issues found in report.
// If dryRun is true no changes are made; actions are described instead.
func Fix(ctx context.Context, ws *workspace.Workspace, report *Report, dryRun bool) []FixResult {
	var results []FixResult
	for i := range report.Categories {
		results = append(results, fixCategory(ctx, ws, &report.Categories[i], dryRun)...)
	}
	return results
}

// FixCategory runs auto-fix actions for issues in a single category.
func FixCategory(ctx context.Context, ws *workspace.Workspace, cat *CategoryReport, dryRun bool) []FixResult {
	return fixCategory(ctx, ws, cat, dryRun)
}

func fixCategory(ctx context.Context, ws *workspace.Workspace, cat *CategoryReport, dryRun bool) []FixResult {
	var results []FixResult
	switch cat.Name {
	case "Git":
		results = append(results, fixOrphanedWorktrees(ctx, ws, cat, dryRun)...)
	case "Workspace":
		results = append(results, fixWorkspace(ws, cat, dryRun)...)
	}
	return results
}

// fixOrphanedWorktrees removes orphaned git worktrees.
func fixOrphanedWorktrees(ctx context.Context, ws *workspace.Workspace, cat *CategoryReport, dryRun bool) []FixResult {
	var results []FixResult
	for _, item := range cat.Items {
		if item.Name != "orphaned worktree" {
			continue
		}
		path := item.Message
		action := fmt.Sprintf("git worktree remove --force %q", path)
		if dryRun {
			results = append(results, FixResult{Action: action, Success: true, Message: "[dry-run]"})
			continue
		}
		cmd := exec.CommandContext(ctx, "git", "-C", ws.RootDir, "worktree", "remove", "--force", path) //nolint:gosec // G204: path comes from git worktree list output
		if err := cmd.Run(); err != nil {
			results = append(results, FixResult{Action: action, Success: false, Message: err.Error()})
		} else {
			results = append(results, FixResult{Action: action, Success: true, Message: "removed"})
		}
	}
	return results
}

// fixWorkspace re-creates missing workspace directories.
func fixWorkspace(ws *workspace.Workspace, cat *CategoryReport, dryRun bool) []FixResult {
	var results []FixResult
	for _, item := range cat.Items {
		if item.Severity != SeverityFail && item.Severity != SeverityWarn {
			continue
		}
		if item.Name != "agents/" && item.Name != "roles/" {
			continue
		}

		var dir string
		switch item.Name {
		case "agents/":
			dir = ws.AgentsDir()
		case "roles/":
			dir = ws.RolesDir()
		}

		action := fmt.Sprintf("mkdir -p %s", dir)
		if dryRun {
			results = append(results, FixResult{Action: action, Success: true, Message: "[dry-run]"})
			continue
		}
		if err := os.MkdirAll(dir, 0750); err != nil {
			results = append(results, FixResult{Action: action, Success: false, Message: err.Error()})
		} else {
			results = append(results, FixResult{Action: action, Success: true, Message: "created"})
		}
	}
	return results
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// formatDuration formats a duration in human-readable form (e.g. "2h15m").
func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
