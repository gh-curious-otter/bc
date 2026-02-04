# Audit: Dead Code and Unused Features (bc-34b.10)

## Summary

Inventory of dead code, unused exports, and config-defined features that are not used in the codebase. Use this as a cleanup list.

---

## 1. Config: Unused

### Cost tracking (`[costs]` / `config.Costs`)

- **Defined in:** `config.toml` and `config/config.go`: `enabled`, `limit`, `warn_threshold`.
- **Used:** Nowhere. No code reads `config.Costs` to enforce limits or log warnings.
- **Action:** Either wire cost tracking into agent/session logic (and document), or remove from config and doc as “future use”.

### Other config

- **`config.Name`**, **`config.Version`**: Defined; usage not audited in full (may appear in version/help).
- **`config.Agent.CoordinatorName`**, **`config.Agent.WorkerPrefix`**: Used for naming; confirm no dead paths.
- **`config.Agents`**: Used in `pkg/agent` for `SetAgentByName`. OK.
- **`config.Tui`**: `RefreshInterval` used in home TUI; `Theme` may be unused (verify TUI theme loading).
- **`config.Roles`**: Prompt files and permissions; used by design. OK.

---

## 2. CLI: Unused flags

### `--verbose` / `-v`

- **Defined:** `rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")` in `internal/cmd/root.go`.
- **Used:** No command calls `cmd.Flags().GetBool("verbose")` or equivalent. Flag is effectively dead.
- **Action:** Either implement verbose logging behind this flag or remove it.

---

## 3. Workspace registry

- **Package:** `pkg/workspace/registry.go` (Registry, LoadRegistry, Register, Unregister, Touch, Prune, List, Find).
- **Used:** `internal/cmd/home.go` (LoadRegistry, Prune, List), `internal/cmd/init.go` (LoadRegistry, Register, Save). In use, not dead.

---

## 4. Stub / minimal implementations

- **`internal/cmd/example.go`**: Example/demo command; intentional stub.
- **`internal/cmd/dashboard.go`**: Dashboard command; verify if used or legacy.
- **`internal/cmd/ui.go`**: UI/demo; verify if used or legacy.

No other obvious stubs identified without deeper call-graph analysis.

---

## 5. Exports to review (possible dead public API)

- **`pkg/agent`:** `CanCreateRole`, `HasCapability`, `RoleLevel`, `IsLeaf`, `Level`, `ListChildren`, `ListDescendants`, `GetParent`, `ListByRole` — used by hierarchy/spawn logic; keep unless proven unused.
- **`pkg/tmux`:** `NewDefaultManager`, `CreateSession` (vs `CreateSessionWithEnv`) — may be unused; run static analysis or tests to confirm.
- **`pkg/workspace`:** `Unregister`, `Touch`, `Find` — confirm call sites; may be used by future or external code.

Recommend running `staticcheck` and/or `golangci-lint` with `unused` to get a full list of unused exports.

---

## 6. Cleanup list (actionable)

| Item | Location | Action |
|------|----------|--------|
| Cost tracking config | config.toml, config.go | Use it or remove/document as future |
| `--verbose` / `-v` | internal/cmd/root.go | Use for logging or remove |
| TUI theme from config | config.Tui.Theme | Verify theme is applied; if not, use or remove |
| Unused tmux/workspace helpers | pkg/tmux, pkg/workspace | Run linters and remove or document |

---

*Audit completed for bead bc-34b.10. Branch: bc-34b.10.*
