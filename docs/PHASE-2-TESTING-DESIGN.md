# Phase 2: Testing Infrastructure - Technical Design Document

**Epic:** #678 (Comprehensive BC UI/UX Enhancement Plan)
**Issues:** #671 (Automated Testing), #675 (Real-time UX/CLI Testing)
**Owner:** vibrant-cheetah agent
**Status:** Design Phase
**Target:** 80%+ test coverage with automated CI/CD validation
**Timeline:** Weeks 1-2 of Phase 2

---

## EXECUTIVE SUMMARY

This document defines the testing infrastructure for bc CLI/TUI with two parallel tracks:

1. **Automated Test Suite** (CI/CD integrated): Unit/integration tests with 80%+ coverage
2. **Real-time Testing Agents** (#675): Dedicated UX engineers running continuous manual testing

The combined approach ensures both regression prevention (automated) and UX validation (manual).

---

## SECTION 1: ANALYSIS & FINDINGS

### 1.1 Current Test Coverage Status

| Category | Metric | Status |
|----------|--------|--------|
| **Test Files** | 9 files | 919 lines of tests |
| **Source Code** | ~7,844 lines | Across 40+ files |
| **Current Coverage** | 11.7% | Components: 37.5%, Hooks: 8%, Views: 0% |
| **Target Coverage** | 80%+ | +1,500-2,000 lines of tests needed |
| **Test Framework** | Bun test runner + ink-testing-library | Established |
| **CI Integration** | GitHub Actions (tui-test job) | Partial |

### 1.2 What Components Need Testing?

**CRITICAL (High Impact, not tested):**
- ✅ **Hooks (Data Layer)**: 12 hooks, 92% untested
  - usePolling.ts (300 lines) - Core infrastructure
  - useDashboard.ts (220 lines) - Dashboard metrics
  - useChannels.ts (124 lines) - Channel fetching
  - useAgents.ts (120 lines) - Agent status
  - 8 other hooks (800+ lines)

- ✅ **Views (Full Pages)**: 9 views, 0% tested
  - Dashboard.tsx (310 lines) - Main view
  - CostDashboard.tsx (200 lines) - Cost analytics
  - MessageHistory.tsx (200 lines) - Message scrolling
  - ProcessesView.tsx (250 lines) - Process list
  - TeamsView.tsx (220 lines) - Team management
  - AgentDetailView.tsx (110 lines) - Agent details
  - DemonsView.tsx (248 lines) - Scheduled tasks
  - ChannelsView.tsx (290 lines) - Channels (partially fixed in Phase 1)
  - AgentsView.tsx (112 lines) - Agent list

- ✅ **Services**: bc.ts (300+ lines) - CLI command execution
  - Command spawning and error handling
  - JSON parsing and validation
  - Timeout management
  - Environment setup

**IMPORTANT (Medium Priority, partially tested):**
- ✅ **Complex Components**: MessageInput, DataTable, MentionAutocomplete (500+ lines)
- ✅ **Navigation**: FocusContext, NavigationContext, useKeyboardNavigation (400+ lines)
- ⚠️ **Simple Components**: Mostly tested but could use edge cases

### 1.3 Coverage Gap Analysis

```
Current State:
├── Tested: Theme system, simple presentational components (11.7%)
├── Critical Gap: Data fetching (hooks) - 92% untested
├── Critical Gap: Views (full page) - 100% untested
├── Critical Gap: Services (CLI execution) - 100% untested
├── Critical Gap: Navigation/State - 75% untested
└── Integration: 0% (multi-view workflows untested)

Target State (80%+):
├── All hooks: 95%+ coverage
├── All views: 70%+ coverage
├── All services: 100% coverage
├── All navigation: 90%+ coverage
├── Integration tests: 50%+ (critical workflows)
└── Total: 80%+ across entire TUI codebase
```

### 1.4 Test Infrastructure Gaps

**MISSING CRITICAL INFRASTRUCTURE:**

1. **Test Utilities** (0 lines)
   - No `renderWithProviders()` helper → providers must be manually wrapped
   - No `mockBcService()` → bc service would spawn real processes
   - No keyboard simulation → keybind tests must be skipped
   - No fixture generators → each test creates mock data from scratch

2. **Fixture Data** (0 lines)
   - No realistic mock agents/channels/demons
   - No cost data fixtures
   - No process listings
   - No error scenario fixtures

3. **CI/CD Validation** (Partial)
   - ✅ GitHub Actions runs tests (tui-test job)
   - ❌ No coverage reporting/thresholds
   - ❌ No coverage badges/history
   - ❌ No blocking coverage gates (e.g., >80% required)
   - ❌ No HTML coverage reports
   - ❌ No coverage diff in PR comments

4. **CLI Testing** (Not implemented)
   - ❌ No automated CLI command testing
   - ❌ No JSON output schema validation
   - ❌ No error scenario testing
   - ❌ No performance benchmarking

5. **Bun Compatibility Issues**
   - ❌ keybind-focus-integration.test.tsx uses jest.fn() (not available in Bun)
   - ❌ No Bun mock equivalent documentation
   - ❌ useInput hook tests must be skipped (TTY limitation)

### 1.5 Answers to Key Questions

**Q1: What TUI components need testing?**
- ALL 9 views need at least 70% coverage (2,400 lines of code)
- ALL 12 hooks need 95%+ coverage (1,100 lines of code)
- Complex components need 90%+ coverage (900 lines)
- Navigation/state management needs 90% coverage (400 lines)
- Service layer needs 100% coverage (300 lines)

**Q2: What's the current test coverage?**
- Measured: 11.7% (919 lines of tests / 7,844 lines of code)
- By category: Components 37%, Hooks 8%, Views 0%, Navigation 25%, Services 0%
- Biggest gap: Hooks (92% untested), Views (100% untested), Services (100% untested)

**Q3: What's missing?**
- Test utilities library (~200 lines needed)
- Fixture data generators (~150 lines needed)
- 1,500+ lines of new test code
- CI/CD coverage integration
- CLI testing framework
- Bun compatibility fixes

**Q4: What's the test infrastructure gap?**
- No helper functions for common patterns
- No mock bc service implementation
- No fixture factories
- No CI coverage validation
- No coverage reporting/history
- No CLI testing infrastructure

---

## SECTION 2: TECHNICAL DESIGN

### 2.1 Testing Framework Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    TUI Testing Architecture                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Layer 1: Test Infrastructure (NEW)                       │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │ • Test Utilities (testUtils.tsx)                         │   │
│  │ • Fixture Generators (fixtures/)                         │   │
│  │ • Mock Service (mocks/)                                  │   │
│  │ • Setup/Teardown (setupTests.ts)                         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              ↓                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Layer 2: Testing Libraries (EXISTING)                    │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │ • ink-testing-library (component rendering)             │   │
│  │ • Bun test runner (execution)                           │   │
│  │ • React Testing Library patterns (queries)              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              ↓                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Layer 3: Test Suites (NEW - 1500+ lines)                 │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │ Unit Tests:                                              │   │
│  │ • src/__tests__/unit/hooks/ (200+ lines)               │   │
│  │ • src/__tests__/unit/components/ (300+ lines)          │   │
│  │ • src/__tests__/unit/services/ (200+ lines)            │   │
│  │                                                          │   │
│  │ Integration Tests:                                       │   │
│  │ • src/__tests__/integration/views/ (400+ lines)        │   │
│  │ • src/__tests__/integration/workflows/ (300+ lines)    │   │
│  │                                                          │   │
│  │ Snapshot Tests:                                          │   │
│  │ • src/__tests__/snapshots/ (200+ lines)                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              ↓                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Layer 4: CI/CD Pipeline (ENHANCED)                       │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │ • GitHub Actions (tui-tests.yml - NEW)                  │   │
│  │ • Coverage reporting (codecov integration)              │   │
│  │ • Coverage gates (minimum 80%)                          │   │
│  │ • Performance benchmarking                              │   │
│  │ • Blocking checks (fail on <80%)                        │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              ↓                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Layer 5: Real-time Testing Agents (#675)                 │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │ • UX Engineer #1: TUI manual testing                     │   │
│  │ • UX Engineer #2: CLI manual testing                     │   │
│  │ • Blocking Detection: `bc report stuck`                 │   │
│  │ • Test Reports: .bc/test-reports/                       │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Test Framework Specifications

**Framework Choice: Bun Test Runner + ink-testing-library**

✅ **Why Bun:**
- Already integrated (package.json has test scripts)
- Fast execution (~10x faster than Jest)
- Native TypeScript support
- Simpler API (no need for jest.fn() workarounds)
- Existing CI/CD uses Bun 1.1.0

✅ **Why ink-testing-library:**
- Purpose-built for Ink terminal components
- Similar API to React Testing Library (familiar)
- Perfect for terminal UI testing
- Supports lastFrame() snapshots for visual regression

**Testing Approach: 3-Tiered**

```typescript
// Tier 1: Unit Tests (Hooks, Simple Components)
test('useChannels fetches and returns channels', async () => {
  const { result } = renderHook(() => useChannels());
  await waitFor(() => expect(result.current.data).toBeDefined());
});

// Tier 2: Component Tests (Complex Components, Views)
test('ChannelsView displays channel list', () => {
  const { lastFrame } = render(
    <ChannelsView disableInput />,
    { wrapper: TestProviders }
  );
  expect(lastFrame()).toContain('channels');
});

// Tier 3: Integration Tests (Multi-view Workflows)
test('User can navigate from channels to messages', async () => {
  const { lastFrame } = render(
    <App />,
    { wrapper: TestProviders }
  );
  // Simulate navigation and verify state changes
});
```

### 2.3 Coverage Target & Strategy

**Target: 80%+ Coverage**

```
Coverage Breakdown by Component:
├── Hooks: 95%+ (critical data layer)
├── Components: 85%+ (UI layer)
├── Views: 70%+ (full page workflows)
├── Services: 100% (no untested code paths)
├── Navigation: 90%+ (state management)
└── Overall: 80%+ (minimum acceptable)

Exclusions (don't count toward coverage):
├── dist/ (compiled output)
├── __tests__/ (test code itself)
├── index.ts files (re-exports)
└── .d.ts files (type definitions)
```

**How to Achieve:**

1. **Phase 1 (Weeks 1-2): Foundation**
   - Create test utilities library (~200 lines)
   - Create fixture generators (~150 lines)
   - Fix Bun compatibility issues
   - Add setupTests.ts configuration
   - Expected coverage: 15-20%

2. **Phase 2 (Weeks 3-4): Data Layer**
   - Test all 12 hooks (400+ lines of tests)
   - Test service layer (200+ lines of tests)
   - Expected coverage: 35-40%

3. **Phase 3 (Weeks 5-6): Component Layer**
   - Test all components (500+ lines of tests)
   - Test navigation (200+ lines of tests)
   - Expected coverage: 60-65%

4. **Phase 4 (Week 7): View & Integration Layer**
   - Test all views (400+ lines of tests)
   - Add integration tests (200+ lines)
   - Expected coverage: 75-80%

5. **Phase 5 (Week 8): Polish & Optimization**
   - Snapshot testing for visual regression
   - Edge case coverage
   - Performance optimization
   - Expected coverage: 80%+

### 2.4 CI/CD Integration

**New GitHub Actions Workflow: `.github/workflows/tui-tests.yml`**

```yaml
name: TUI Tests & Coverage

on:
  push:
    branches: [main, feature/**]
    paths:
      - 'tui/src/**'
      - '.github/workflows/tui-tests.yml'
  pull_request:
    branches: [main]

jobs:
  test:
    name: TUI Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Bun
        uses: oven-sh/setup-bun@v1
        with:
          bun-version: 1.1.0

      - name: Install Dependencies
        run: cd tui && bun install

      - name: Run Tests
        run: cd tui && bun test --coverage

      - name: Check Coverage
        run: |
          cd tui
          COVERAGE=$(bun test --coverage 2>&1 | grep -o '[0-9]*%' | head -1 | tr -d '%')
          if [ "$COVERAGE" -lt 80 ]; then
            echo "Coverage ${COVERAGE}% is below 80% threshold"
            exit 1
          fi

      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./tui/coverage/coverage.json
          flags: tui
          name: tui-coverage

      - name: Comment Coverage on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const coverage = await getCoverage();
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `📊 **TUI Coverage: ${coverage}%**\n✅ Tests passed`
            });

  typecheck:
    name: TypeScript Check
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v1
      - run: cd tui && bun install && bun run typecheck

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v1
      - run: cd tui && bun install && bun run lint
```

**Coverage Reporting:**
- Codecov integration for historical tracking
- PR comments with coverage %
- Branch coverage comparison
- Detailed coverage reports in artifacts

### 2.5 CLI Testing Infrastructure

**CLI Testing Approach:**

```typescript
// File: src/__tests__/cli/commands.test.ts

import { execSync } from 'child_process';

describe('CLI Commands', () => {
  describe('JSON Output Validation', () => {
    test('bc status --json produces valid JSON', () => {
      const output = execSync('bc status --json').toString();
      const parsed = JSON.parse(output); // Throws if invalid JSON
      expect(parsed).toHaveProperty('agents');
    });

    test('bc channel list --json matches schema', () => {
      const output = execSync('bc channel list --json').toString();
      const data = JSON.parse(output);
      expect(data).toMatchSchema(channelsSchema);
    });
  });

  describe('Error Handling', () => {
    test('Invalid command returns error', () => {
      expect(() => {
        execSync('bc invalid-command');
      }).toThrow();
    });
  });

  describe('Performance', () => {
    test('bc status completes in <1s', () => {
      const start = Date.now();
      execSync('bc status');
      expect(Date.now() - start).toBeLessThan(1000);
    });
  });
});
```

**CLI Testing Coverage:**
- ✅ All major commands (agent, channel, cost, demon, etc.)
- ✅ JSON output validation against schemas
- ✅ Error scenarios (invalid input, missing files)
- ✅ Performance benchmarks (command latency)
- ✅ Integration with latest build

### 2.6 Performance Metrics Strategy

**Tracked Metrics:**

1. **Test Execution Performance**
   - Total test runtime (target: <10s for full suite)
   - Per-file execution time
   - Slowest tests (alert if >1s)

2. **Code Coverage Metrics**
   - Line coverage %
   - Branch coverage %
   - Function coverage %
   - Uncovered lines (prioritize)

3. **Visual Regression Detection**
   - Snapshot test comparisons
   - Visual diff reports
   - Pixel-perfect component rendering

4. **CLI Performance Benchmarks**
   - Command latency (must be <1s)
   - Memory usage
   - Process creation overhead

**Implementation:**

```typescript
// File: src/__tests__/utils/performanceMonitor.ts

export class PerformanceMonitor {
  private metrics: Map<string, number> = new Map();

  measure(name: string, fn: () => void): number {
    const start = Date.now();
    fn();
    const duration = Date.now() - start;
    this.metrics.set(name, duration);
    return duration;
  }

  getReport(): PerformanceReport {
    return {
      total: Array.from(this.metrics.values()).reduce((a, b) => a + b),
      slow: Array.from(this.metrics.entries())
        .filter(([_, time]) => time > 1000)
        .sort((a, b) => b[1] - a[1])
    };
  }
}
```

---

## SECTION 3: IMPLEMENTATION BREAKDOWN

### 3.1 Subtask 1: TUI Testing Framework Setup

**Objective:** Create test infrastructure foundation with utilities and fixtures
**Duration:** 3-4 days
**Owner:** vibrant-cheetah (primary), young-narwhal (support)

**Deliverables:**

1. **Test Utilities Library** (`src/__tests__/utils/testUtils.tsx`)
   ```typescript
   // Helpers needed:
   - renderWithProviders(component, options)
   - mockBcService(responses)
   - createMockAgent(overrides)
   - createMockChannel(overrides)
   - simulateKeypress(key)
   - waitForElement(predicate)
   ```
   **Effort:** 3-4 hours
   **Tests:** 10+ unit tests validating each utility

2. **Fixture Data Generators** (`src/__tests__/fixtures/`)
   ```typescript
   // Files:
   - agents.ts (agent factory with states)
   - channels.ts (channel factory with messages)
   - demons.ts (demon task factory)
   - costs.ts (cost data factory)
   - processes.ts (process factory)
   - teams.ts (team factory)
   ```
   **Effort:** 2-3 hours
   **Tests:** 5+ unit tests per factory

3. **Setup/Teardown Configuration** (`src/__tests__/setup.ts`)
   ```typescript
   // Configure:
   - Global test timeout
   - Mock timers (if needed)
   - Provider setup
   - Cleanup routines
   ```
   **Effort:** 1 hour

4. **Mock Service Implementation** (`src/__tests__/mocks/bc.ts`)
   ```typescript
   // Mock bc service to:
   - Return fixture data
   - Simulate delays
   - Trigger errors
   - Track call history
   ```
   **Effort:** 3-4 hours

5. **Bun Compatibility Fixes**
   - Replace jest.fn() with Bun mocks in keybind-focus-integration.test.tsx
   - Document Bun testing patterns for team
   **Effort:** 2 hours

**Acceptance Criteria:**
- [ ] All 4 utility functions work and are tested
- [ ] All 6 fixture factories generate realistic data
- [ ] No jest.fn() references in codebase
- [ ] Mock bc service passes 10+ validation tests
- [ ] New tests don't break existing 919 lines of tests
- [ ] TypeScript compilation succeeds with 0 errors

**PR Preview:** "test(tui): Add testing framework foundation with utilities and fixtures"

---

### 3.2 Subtask 2: Data Layer Testing (Hooks & Services)

**Objective:** Achieve 95%+ coverage of data fetching and service layer
**Duration:** 4-5 days
**Owner:** vibrant-cheetah (primary), young-narwhal (support on CLI tests)

**Deliverables:**

1. **Hook Tests** (`src/__tests__/unit/hooks/`)
   - `useChannels.test.ts` - 20+ tests (fetching, polling, errors)
   - `useAgents.test.ts` - 20+ tests (state transitions, updates)
   - `useDashboard.test.ts` - 20+ tests (metrics aggregation)
   - `usePolling.test.ts` - 15+ tests (interval management)
   - `useCosts.test.ts` - 10+ tests (data aggregation)
   - `useDemons.test.ts` - 15+ tests (scheduling)
   - `useProcesses.test.ts` - 10+ tests (process listing)
   - `useTeams.test.ts` - 10+ tests (team data)
   - `useStatus.test.ts` - 10+ tests (workspace status)
   - Other hooks: 5-10 tests each

   **Total:** 150+ tests, 500+ lines
   **Effort:** 3-4 days

2. **Service Layer Tests** (`src/__tests__/unit/services/`)
   - `bc.test.ts` - 30+ tests
     - Command execution (success/failure)
     - JSON parsing and validation
     - Timeout handling
     - Error handling (stderr parsing)
     - Environment variable override

   **Total:** 30+ tests, 200+ lines
   **Effort:** 1-2 days

3. **Integration with CLI** (with young-narwhal)
   - JSON schema validation tests
   - Command output format tests
   - Error message consistency tests

   **Total:** 20+ tests, 150+ lines
   **Effort:** 1 day

**Acceptance Criteria:**
- [ ] 95%+ line coverage for all hooks
- [ ] 100% coverage for bc.ts service
- [ ] All edge cases tested (errors, timeouts, invalid data)
- [ ] Mocked bc service used throughout (no real command execution)
- [ ] All tests pass with <15s total runtime
- [ ] Test file organization mirrors source file structure

**Expected Coverage Improvement:** 11.7% → 40-45%

**PR Preview:** "test(tui): Add comprehensive hook and service layer tests (+30% coverage)"

---

### 3.3 Subtask 3: Component & Integration Testing + CI Setup

**Objective:** Test components and views, integrate into CI/CD, establish coverage gates
**Duration:** 5-6 days
**Owner:** vibrant-cheetah (primary), noble-vulture (review)

**Deliverables:**

1. **Component Tests** (`src/__tests__/unit/components/`)
   - Simple components (ErrorDisplay, LoadingIndicator, Panel): 5+ tests each
   - Complex components (MessageInput, DataTable): 20+ tests each
   - Visual components (ChatMessage, Reaction): 10+ tests each
   - All components: 300+ tests, 500+ lines

   **Effort:** 2-3 days

2. **View Tests** (`src/__tests__/integration/views/`)
   - Dashboard.test.tsx: 25+ tests (metrics, rendering)
   - ChannelsView.test.tsx: 30+ tests (with Phase 1 fixes)
   - MessageHistory.test.tsx: 20+ tests (scrolling, message display)
   - AgentsView.test.tsx: Expand existing tests to 20+
   - Other views: 15-20 tests each
   - All views: 200+ tests, 400+ lines

   **Effort:** 2-3 days

3. **Integration Tests** (`src/__tests__/integration/workflows/`)
   - Navigation workflow: 15+ tests
   - Message sending workflow: 15+ tests
   - Focus state synchronization: 10+ tests
   - Keyboard navigation: 15+ tests
   - Multi-view navigation: 15+ tests
   - Total: 70+ tests, 300+ lines

   **Effort:** 2 days

4. **Snapshot Tests** (`src/__tests__/snapshots/`)
   - Visual regression detection for major views
   - Dashboard snapshot
   - Key views snapshots
   - Total: 10-15 snapshots

   **Effort:** 1 day

5. **CI/CD Integration**
   - Create `.github/workflows/tui-tests.yml` (80-100 lines)
   - Add coverage validation (>80% gate)
   - Codecov integration
   - PR comment with coverage report
   - Build artifacts for coverage reports

   **Effort:** 2 days

6. **Coverage Report Dashboard**
   - Bun coverage integration
   - Coverage history tracking
   - Report generation script

   **Effort:** 1 day

**Acceptance Criteria:**
- [ ] 85%+ coverage for all components
- [ ] 70%+ coverage for all views
- [ ] 10-15 snapshot tests created
- [ ] CI/CD workflow runs tests and checks coverage
- [ ] Coverage <80% blocks merge (enforced in CI)
- [ ] PR comments show coverage %
- [ ] Codecov historical tracking works
- [ ] All tests pass in CI/CD environment

**Expected Coverage Improvement:** 40-45% → 80%+

**PR Preview:** "test(tui): Add component, view, and integration tests with CI/CD coverage gates (+40% coverage)"

---

### 3.4 Success Metrics & Checkpoints

**Weekly Checkpoints:**

| Week | Subtask | Target | Checkpoint |
|------|---------|--------|-----------|
| 1 | Subtask 1 (Foundation) | 15-20% coverage | Test utils working, fixtures generated |
| 2 | Subtask 2 (Data Layer) | 40-45% coverage | All hooks tested, services 100% covered |
| 2 | Subtask 3 (Components) | 70-75% coverage | Components + views tested |
| 2 | Subtask 3 (CI/CD) | 80%+ coverage | CI integration live, coverage gates enforced |

**Success Definition:**
- ✅ 80%+ line coverage across TUI codebase
- ✅ All PRs must maintain >80% coverage
- ✅ CI/CD blocks merges <80% coverage
- ✅ No manual testing skipped (`.skip()` tests removed)
- ✅ Test suite runs in <15 seconds
- ✅ Visual regression detection with snapshots
- ✅ Real-time testing agents (#675) can run confidently with test coverage

---

## SECTION 4: REAL-TIME TESTING AGENTS (#675)

### 4.1 Parallel Track: Agent-Based Testing

While automated tests run in CI/CD, two dedicated agents (#675) will perform:

**UX Engineer #1: TUI Testing**
- Test all views: Dashboard, Channels, Agents, Costs, Demons, Processes, Teams
- Validate keyboard navigation
- Test with different terminal sizes
- Report visual issues via `bc report stuck`
- Continuous testing loop

**UX Engineer #2: CLI Testing**
- Test all commands: agent, channel, cost, demon, role, etc.
- Validate help output
- Test error scenarios
- Test with test workspace
- Report CLI issues via `bc report stuck`

**Blocking Detection:**
```bash
bc report stuck \
  --reason "TUI freezes when pressing j in channels" \
  --reproduction "Start bc home, go to channels, press j multiple times" \
  --severity critical \
  --blocks testing
```

This creates automatic GitHub issues and stops test execution until resolved.

### 4.2 Integration Between Automated & Manual Testing

```
┌─────────────────────────────────────────────────────────┐
│         Testing Feedback Loop                            │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Automated Tests (CI/CD)                                │
│  ├─ Runs on every push                                 │
│  ├─ Validates 80%+ coverage                            │
│  ├─ Checks JSON output schemas                         │
│  └─ Reports: PASS/FAIL coverage %                      │
│        ↓                                                │
│  Manual Tests (UX Agents)                               │
│  ├─ Run continuously on main branch                    │
│  ├─ Test UX, visual appearance, interactions          │
│  ├─ Report visual issues: `bc report stuck`           │
│  └─ Reports: Issues found, UX feedback                │
│        ↓                                                │
│  Issues & Reports                                       │
│  ├─ Automated: 80% coverage gate failure               │
│  ├─ Manual: Visual regressions, UX problems            │
│  └─ Review: noble-vulture quality gates                │
│        ↓                                                │
│  Resolution & Merge                                     │
│  ├─ Fix tests/coverage                                 │
│  ├─ Fix visual issues                                  │
│  └─ Deploy to main                                     │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## SECTION 5: TIMELINE & RESOURCE ALLOCATION

### 5.1 Phase 2 Timeline (Weeks 1-2)

**Week 1:**
- Mon-Tue: Subtask 1 - Test framework foundation (test utils, fixtures, setup)
- Wed-Thu: Subtask 2 start - Hook tests (useChannels, useAgents, etc.)
- Fri: Code review checkpoint, iteration on feedback

**Week 2:**
- Mon-Tue: Subtask 2 continued - Service layer tests, CLI integration
- Wed-Thu: Subtask 3 - Component & view tests
- Fri: Subtask 3 continued - CI/CD setup, coverage gates, final validation

**Deliverables by EOW2:**
1. ✅ PR: Test framework foundation (Subtask 1)
2. ✅ PR: Hook and service tests (Subtask 2)
3. ✅ PR: Component, view, and integration tests (Subtask 3)
4. ✅ 80%+ coverage achieved
5. ✅ CI/CD pipeline live with coverage gates

### 5.2 Resource Allocation

| Resource | Role | Allocation |
|----------|------|-----------|
| vibrant-cheetah | Lead testing design, implement Subtasks 1-3 | 80% (0.8 FTE) |
| young-narwhal | CLI testing, fixtures for backend data | 30% (0.3 FTE) |
| noble-vulture | Quality review, coverage gate validation | 20% (0.2 FTE) |
| Total Team Allocation | | 1.3 FTE equivalent |

### 5.3 Dependencies & Blockers

**No External Blockers:**
- Bun test runner already available ✅
- ink-testing-library already in dependencies ✅
- GitHub Actions already configured ✅
- Phase 1 work merged, can build on it ✅

**Potential Risks:**
- TTY/useInput limitations may require test skipping
- Large fixture data generation may slow test execution
- Codecov integration may require setup

**Mitigation:**
- Mock bc service to avoid TTY issues
- Use lazy fixture generation
- Have fallback local coverage reporting if Codecov fails

---

## SECTION 6: QUALITY GATES & ACCEPTANCE

### 6.1 Phase 2 Success Criteria

**Must Have (Blocking):**
- [ ] 80%+ line coverage across TUI
- [ ] All Subtask 1 deliverables complete
- [ ] All Subtask 2 deliverables complete
- [ ] All Subtask 3 deliverables complete
- [ ] CI/CD pipeline enforces coverage gates
- [ ] No `.skip()` tests in new code
- [ ] TypeScript compilation succeeds
- [ ] All tests pass in CI/CD
- [ ] noble-vulture quality gate approval

**Should Have (Non-blocking):**
- [ ] 90%+ coverage for critical paths
- [ ] Snapshot tests for visual regression detection
- [ ] Coverage history tracking in Codecov
- [ ] Performance benchmarks established
- [ ] PR comments with coverage diffs

**Nice to Have (Future):**
- [ ] HTML coverage reports
- [ ] Branch coverage tracking
- [ ] Performance regression detection
- [ ] Visual diff reports in PRs

### 6.2 Review Checklist for noble-vulture

**Code Quality:**
- [ ] Tests follow Bun/ink-testing-library best practices
- [ ] Fixtures are realistic and maintainable
- [ ] Mock service properly covers all code paths
- [ ] No hardcoded test data in test files
- [ ] Test organization mirrors source structure

**Coverage:**
- [ ] 80%+ coverage verified independently
- [ ] High-risk code (hooks, services) 95%+
- [ ] All branches tested (not just happy path)
- [ ] Error scenarios covered

**CI/CD:**
- [ ] Coverage gates enforced in workflow
- [ ] Tests run consistently in CI environment
- [ ] Coverage diffs show in PR comments
- [ ] Codecov integration working

**Maintainability:**
- [ ] New tests easy to understand and extend
- [ ] Fixtures are DRY (no duplication)
- [ ] Test utilities well documented
- [ ] CI/CD config clear and maintainable

---

## SECTION 7: NEXT STEPS & HANDOFF

### 7.1 Immediate Actions (This Week)

1. **Create implementation subtasks** as GitHub issues:
   - Issue: "Subtask 1: TUI Testing Framework Setup"
   - Issue: "Subtask 2: Data Layer Tests (Hooks & Services)"
   - Issue: "Subtask 3: Component Testing & CI/CD Integration"

2. **Get team alignment:**
   - Share this design doc with young-narwhal and noble-vulture
   - Confirm resource availability
   - Discuss any concerns/blockers

3. **Prepare environment:**
   - Review current test infrastructure
   - Identify any Bun compatibility issues
   - Prepare Codecov setup

4. **Start Subtask 1:**
   - Create test utilities
   - Generate fixtures
   - Fix Bun compatibility issues

### 7.2 Epic #678 Status Update

**Phase 1: ✅ COMPLETE**
- PR #679 merged with 3 quick wins
- Channel message overflow fixed
- Channel descriptions added
- Visual borders standardized

**Phase 2: 🚀 IN PROGRESS**
- Design doc: Complete (this doc)
- Subtasks: Ready for implementation
- Timeline: 2 weeks

**Phase 3-5: 📋 PLANNED**
- Animations, responsive layouts, command palette
- CLI improvements with pkg/ui
- Documentation and issue tracking

---

## APPENDIX A: Test Utilities Reference

### Example: renderWithProviders

```typescript
export function renderWithProviders(
  component: React.ReactElement,
  options?: {
    initialView?: View;
    theme?: ThemeMode;
    focusedArea?: FocusArea;
  }
) {
  return render(
    <ThemeProvider initialTheme={options?.theme}>
      <FocusProvider initialFocus={options?.focusedArea}>
        <NavigationProvider initialView={options?.initialView}>
          {component}
        </NavigationProvider>
      </FocusProvider>
    </ThemeProvider>,
    { disableInput: true }
  );
}
```

### Example: Fixture Generators

```typescript
export function createMockAgent(overrides?: Partial<Agent>): Agent {
  return {
    id: 'agent-1',
    name: 'test-agent',
    role: 'engineer',
    state: 'idle',
    task: 'Test task',
    session: 'session-1',
    workspace: 'test-workspace',
    worktree_dir: '/tmp/worktree',
    memory_dir: '/tmp/memory',
    started_at: '2026-02-15T10:00:00Z',
    updated_at: '2026-02-15T11:00:00Z',
    ...overrides,
  };
}
```

---

## APPENDIX B: References

**Related Issues:**
- Epic #678: Comprehensive BC UI/UX Enhancement Plan
- Issue #671: Testing Infrastructure - Automated CLI & TUI Testing
- Issue #675: Real-time UX and CLI Testing with Blocking Detection
- Phase 1 PR #679: Channel message overflow, descriptions, borders

**Documentation:**
- Bun Testing: https://bun.sh/docs/test/introduction
- ink-testing-library: https://github.com/vadimdemedes/ink-testing-library
- Codecov: https://codecov.io/docs

**Configuration Files:**
- `.github/workflows/tui-tests.yml` (NEW)
- `tui/bunfig.toml` (EXISTING)
- `tui/package.json` (EXISTING)
- `tui/tsconfig.json` (EXISTING)

---

**Document Status:**
- ✅ Design Complete: 2026-02-15
- ⏳ Ready for Subtask Implementation
- 📊 Target Coverage: 80%+
- ⏱️ Timeline: 2 weeks (Weeks 1-2 of Phase 2)

