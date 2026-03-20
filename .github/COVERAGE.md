# Code Coverage Standards

## Overview
bc maintains high code quality through continuous coverage enforcement. All PRs must meet coverage thresholds before merge.

## Coverage Requirements

### Global Minimum: 66.6%
All pull requests must maintain or improve overall code coverage to **66.6% or higher**.

**Current Status**:
- Phase 1: 66.6% (baseline) — **enforced in CI**
- Phase 2 Target: 80%+ — not yet enforced
- Target achieved when: Enough tests added to sustain 80%

### Per-Package Guidelines (Phase 3)

When implemented, these minimums will apply:

| Category | Minimum | Examples |
|----------|---------|----------|
| Critical Infrastructure | 95%+ | git, tmux, agent, demon |
| Core Features | 85%+ | channel, memory, cost, team |
| Support Packages | 80%+ | log, routing, stats, names |
| Integration/Optional | 70%+ | integrations, examples |

## How Coverage Works

### Automatic Enforcement
- **When**: Every PR to main
- **Check**: Coverage threshold in CI
- **Failure**: PR check fails if coverage < 66.6%
- **Resolution**: Add tests until coverage improves

### Measuring Coverage
```bash
# Generate coverage report
make coverage

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

### Coverage Report
```bash
# See total coverage percentage
go tool cover -func=coverage.out | grep total
```

## Phase 2 Enforcement

### CI Gate Details
- **Location**: `.github/workflows/ci.yml`
- **Step**: `make coverage` (includes threshold check)
- **Trigger**: After test run
- **Action**: Verify >= 66.6%
- **Result**: Fail PR if below threshold

### For PR Authors
1. Write code and tests
2. Run `make coverage` locally
3. If below threshold, add tests to improve coverage
4. Submit PR when coverage passes

### For Code Reviewers
1. Check coverage report in CI
2. Verify coverage meets threshold
3. Approve only if CI passes
4. Coverage gate prevents merge if below threshold

## Phase 3 Improvements

Future enhancements will include:
- Per-package coverage tracking
- Breakdown by package in PR comments
- Trend analysis (week over week)
- Coverage regression detection
- Optional --strict-coverage mode for higher standards

---

**Status**: Phase 1 enforcement live (66.6%)
**Last Updated**: 2026-03-21
**Next Review**: When coverage consistently exceeds 80%
