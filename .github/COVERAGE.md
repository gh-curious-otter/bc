# Code Coverage Standards

## Overview
bc maintains high code quality through continuous coverage enforcement. All PRs must meet coverage thresholds before merge.

## Coverage Requirements

### Global Minimum: 80%+
All pull requests must maintain or improve overall code coverage to **80% or higher**.

**Current Status**:
- Phase 1: 66.6% (baseline)
- Phase 2 Target: 80%+
- Target achieved when: New PRs reach 80%

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
- **Failure**: PR check fails if coverage < 80%
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
- **Step**: "Check coverage threshold (80%+)"
- **Trigger**: After `make coverage`
- **Action**: Parse coverage.out and verify >= 80%
- **Result**: Fail PR if below threshold

### For PR Authors
1. Write code and tests
2. Run `make test` and `make coverage` locally
3. Check coverage: `go tool cover -func=coverage.out | grep total`
4. If < 80%, add tests to improve coverage
5. Submit PR when coverage >= 80%

### For Code Reviewers
1. Check coverage report in CI
2. Verify coverage meets 80%+ threshold
3. Approve only if CI passes
4. Coverage gate prevents merge if below threshold

## Phase 3 Improvements

Future enhancements will include:
- Per-package coverage tracking
- Breakdown by package in PR comments
- Trend analysis (week over week)
- Coverage regression detection
- Optional --strict-coverage mode for higher standards

## Questions?

- See `.bc/COVERAGE_STANDARDS.md` for internal details
- Check `#eng` channel for discussions
- Review swift-puma test reports for analysis

---

**Status**: Phase 2 enforcement live
**Last Updated**: 2026-02-15
**Next Review**: When Phase 2 PRs start submitting
