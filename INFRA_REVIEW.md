# Infrastructure Review — bc

**Date:** 2026-03-21 (revised)
**Repo:** gh-curious-otter/bc
**Codebase:** Go 31K LOC + TypeScript TUI/Web/Landing
**Issues reviewed:** 1,061 (128 open, 933 closed)
**PRs reviewed:** 23 infra-touching PRs

## Context

bc is a CLI-first AI agent orchestration system. Started Feb 2026, it went through a major server-first architecture pivot (bcd daemon + web UI) in mid-March. The project is pre-release, local-only by design, built partly by its own AI agents. Single developer (rpuneet) with agent-assisted contributions.

## Current State

### What Works
- CI: lint -> fast test -> TUI -> build gate (~90s for PRs, full suite on main)
- 0 golangci-lint issues (115 fixed in PR #2166)
- GoReleaser for Linux/macOS releases with Homebrew tap
- 11 Docker images (1 base + 7 agent providers + bcd + bcdb + deprecated root)
- Cloudflare Pages for landing site, GitHub Pages for docs
- Solid crypto (AES-256-GCM, PBKDF2-600k)
- slog structured logging, strict error handling

### What's Broken
- `scripts/install.sh` will 404 -- expects raw binaries, releases produce archives
- GoReleaser `mode: replace` races with macOS release job -- can delete macOS assets
- Windows `nosqlite` build tag doesn't exist in Go code -- Windows build will fail
- `docker/Dockerfile.base` has Go 1.24.1, project requires 1.25.1+
- `docker/bcdb/01-init.sql` is orphaned -- conflicts with `init.sql` (different schemas)
- E2E tests in `internal/cmd` require running bcd -- can't run in CI

## Issue Status (infra #2044-#2104)

| Issue | Title | Status |
|-------|-------|--------|
| #2051 | Hardcoded POSTGRES_PASSWORD | NOT FIXED |
| #2052 | HTTP body size limits | NOT FIXED |
| #2053 | Secret scanning (gitleaks) | PARTIALLY -- added but non-blocking |
| #2054 | Go version inconsistency | PARTIALLY -- CI fixed, Dockerfile.base still 1.24.1 |
| #2055 | Dependency audit (govulncheck) | PARTIALLY -- added but non-blocking |
| #2056 | Rate limiting | NOT FIXED |
| #2057 | CI caching | CLOSED -- Go cache done |
| #2058 | CORS restriction | NOT FIXED |
| #2059 | .env.example | NOT FIXED |
| #2062 | LICENSE file | CLOSED (not_planned) -- licensing TBD |
| #2089 | SQLite stores bypass pkg/db | NOT FIXED -- 5 stores still direct |
| #2102 | Go stdlib vulns | NOT FIXED -- still on 1.25.1 |
| #2103 | Duplicate release job | CLOSED -- fixed in PR #2163 |
| #2104 | Cost store context.Background() | NOT FIXED -- 23 instances |

## Findings by Area

### CI/CD

| # | Finding | Severity | Action |
|---|---------|----------|--------|
| 1 | `make build-release` only builds `bc`, not `bcd` -- server compilation errors undetected | HIGH | Add `bcd` build to CI |
| 2 | Web UI (`web/`) completely absent from CI | HIGH | Add build + lint job |
| 3 | `go generate` drift undetected -- config.toml changes without regen pass CI | HIGH | Add gen-check step |
| 4 | No `gofmt` enforcement in CI (pre-commit hook only) | MEDIUM | Add to lint job |
| 5 | No concurrency controls -- parallel CI runs waste resources | MEDIUM | Add concurrency groups |
| 6 | No job timeouts -- default 6hr | MEDIUM | Add 15min timeouts |
| 7 | All third-party actions pinned to major version only, not SHA | MEDIUM | Pin to SHA |
| 8 | No `dependabot.yml` for automated dep updates | MEDIUM | Add config |
| 9 | TUI test `continue-on-error: true` -- failures invisible | LOW | Fix tests, remove flag |
| 10 | No Bun cache in TUI job | LOW | Add cache step |

### Docker

| # | Finding | Severity | Action |
|---|---------|----------|--------|
| 1 | `Dockerfile.bcdb:7` hardcoded `POSTGRES_PASSWORD=bc` | CRITICAL | Use runtime env var |
| 2 | `Dockerfile.base:7` Go 1.24.1, needs 1.25.1+ | CRITICAL | Update version |
| 3 | `docker/bcdb/01-init.sql` orphaned, conflicts with `init.sql` | HIGH | Delete orphan |
| 4 | No HEALTHCHECK in any Dockerfile | HIGH | Add to bcd, bcdb |
| 5 | `Dockerfile.bcd` runs as root | HIGH | Document as intentional (Docker socket) |
| 6 | `.dockerignore` missing `.git/` -- bloated build context | HIGH | Add `.git/` |
| 7 | `oven/bun:latest` in 6 Dockerfiles -- unpinned | MEDIUM | Pin version |
| 8 | No version pinning on any agent CLI tool | MEDIUM | Pin versions |
| 9 | `curl | bash` in Dockerfile.claude and Dockerfile.base | MEDIUM | Verify checksums |
| 10 | Host networking by default -- no container isolation | MEDIUM | Document trade-off |
| 11 | Deprecated `Dockerfile.agent` at root still exists | LOW | Delete |
| 12 | No docker-compose.yml | LOW | Create for local dev |

### Build & Release

| # | Finding | Severity | Action |
|---|---------|----------|--------|
| 1 | `scripts/install.sh` URLs don't match release archive names -- will 404 | BUG | Fix URL pattern |
| 2 | GoReleaser `mode: replace` races with macOS job -- can delete assets | BUG | Change to `append` or add ordering |
| 3 | Windows `nosqlite` build tag doesn't exist in code | BUG | Remove Windows build or add tag |
| 4 | CI installs `gcc-aarch64-linux-gnu` but GoReleaser skips linux/arm64 | WASTE | Remove install or enable arm64 |
| 5 | Stale `scripts/homebrew/bc.rb` conflicts with goreleaser-generated formula | REDUNDANCY | Delete manual formula |
| 6 | macOS archives have platform-named binaries, goreleaser has `bc` | INCONSISTENCY | Standardize |
| 7 | Split checksum files (checksums.txt + checksums-macos.txt) | LOW | Merge |
| 8 | `mise.toml` only configures Go, missing bun/golangci-lint/python | LOW | Add tools |

### Security & API

| # | Finding | Severity | Action |
|---|---------|----------|--------|
| 1 | No HTTP body size limits -- OOM via large payloads | HIGH | Add MaxBytesReader |
| 2 | CORS `*` on all interfaces when Docker exposes 0.0.0.0 | MEDIUM | Make configurable |
| 3 | No rate limiting | MEDIUM | Add middleware |
| 4 | 7 Go stdlib CVEs (crypto/tls, net/url, os, crypto/x509) | HIGH | Upgrade to Go 1.25.8 |
| 5 | `--dangerously-skip-permissions` in default config.toml | MEDIUM | Remove from default |
| 6 | 5 SQLite stores bypass pkg/db with weaker settings | MEDIUM | Migrate to pkg/db |
| 7 | 23 `context.Background()` in pkg/cost -- breaks cancellation | MEDIUM | Accept context params |

## Priority Action Plan

### Immediate (bugs that break things)
1. Fix `scripts/install.sh` URL pattern to match release archives
2. Fix GoReleaser `mode: replace` to `mode: append`
3. Remove Windows goreleaser build (nosqlite tag doesn't exist)
4. Update `Dockerfile.base` Go from 1.24.1 to 1.25.1
5. Delete orphaned `docker/bcdb/01-init.sql`

### Week 1 (CI/build gaps)
6. Add `bcd` build verification to CI
7. Add web UI build + lint to CI
8. Add `go generate` drift check
9. Add concurrency groups and job timeouts
10. Add `.git/` to `.dockerignore`

### Week 2 (security hardening)
11. Remove hardcoded POSTGRES_PASSWORD from Dockerfile.bcdb
12. Add `http.MaxBytesReader` to all API handlers
13. Upgrade Go to 1.25.8 (7 stdlib CVEs)
14. Add HEALTHCHECK to bcd and bcdb Dockerfiles
15. Remove `--dangerously-skip-permissions` from default config

### Week 3 (code quality)
16. Migrate 5 SQLite stores to pkg/db
17. Add context.Context to pkg/cost methods
18. Delete stale `scripts/homebrew/bc.rb` and deprecated `Dockerfile.agent`
19. Add `dependabot.yml`
20. Create docker-compose.yml for local dev

## Recurring Patterns (from 1,061 issues + 23 PRs)

1. **Go version drift** -- CI version fell behind go.mod 3 times. No automation.
2. **Coverage threshold instability** -- changed 3 times (80% to 66.6% to 60%). Root cause: E2E tests can't run in CI.
3. **Docker agent reliability** -- 4+ iterative fix PRs. Container lifecycle is complex.
4. **Lint accumulation** -- multiple cleanup waves (ESLint 2831 to 0, golangci-lint 115 to 0). Lint debt accumulates during fast feature sprints.
5. **Architecture pivots** -- major server-first pivot in March. Expect more infrastructure churn as bcd stabilizes.

---
*Generated from automated audit of 1,061 issues, 23 infra PRs, 5 CI workflows, 11 Dockerfiles, and full build system on 2026-03-21.*
