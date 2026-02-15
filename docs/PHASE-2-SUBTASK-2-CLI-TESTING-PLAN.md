# Phase 2 Subtask 2: CLI Testing Implementation Plan

**Author:** young-narwhal
**Status:** Draft (Ready for Subtask 1 Completion)
**Expected Start:** Week 2, Monday (after Subtask 1: 3-4 days)
**Duration:** 4-5 days
**Target Coverage Improvement:** 11.7% → 40-45%

---

## Executive Summary

Subtask 2 focuses on testing the data layer (hooks, services, CLI integration). This document details the **CLI integration testing** portion (20+ tests, 150+ lines) that young-narwhal will lead, working alongside vibrant-cheetah on hook and service layer tests.

The CLI testing layer validates that:
1. All bc commands produce valid JSON
2. JSON output matches expected schemas
3. Error scenarios are handled gracefully
4. Command performance meets SLAs (<1s per command)
5. Environment variables (BC_BIN, BC_ROOT) work correctly

---

## Part 1: CLI Service Layer Tests (100% Coverage Target)

### 1.1 File: `src/__tests__/unit/services/bc.test.ts` (200+ lines, 30+ tests)

**Tests Required:**

#### Group 1: Basic Command Execution (8 tests)
```typescript
describe('execBc - Command Execution', () => {
  test('executes command and returns stdout');
  test('trims whitespace from output');
  test('auto-injects --json flag for json-enabled commands');
  test('does not double-inject --json flag');
  test('rejects on non-zero exit code');
  test('includes stderr in error message');
  test('handles process spawn errors');
  test('kills process on 30s timeout');
});
```

**Key Test Patterns:**
- Mock `spawn` from `child_process`
- Test with `BC_BIN` and `BC_ROOT` environment variables
- Verify stdout/stderr handling
- Verify timeout mechanism (30s)

#### Group 2: JSON Parsing & Error Handling (8 tests)
```typescript
describe('execBcJson - JSON Parsing', () => {
  test('parses valid JSON responses');
  test('throws on invalid JSON with helpful error message');
  test('preserves JSON structure through parse');
  test('handles JSON with special characters');
  test('handles empty JSON arrays');
  test('handles null values in JSON');
  test('handles deeply nested JSON');
  test('provides truncated output on parse error');
});
```

**Key Test Patterns:**
- Test with realistic bc JSON outputs
- Verify error messages include output preview
- Test edge cases (empty arrays, null values, special chars)

#### Group 3: Convenience Methods (14 tests)
```typescript
describe('Convenience Methods', () => {
  // Status
  test('getStatus returns StatusResponse with agents array');

  // Channels
  test('getChannels returns ChannelsResponse with channels array');
  test('getChannelHistory executes bc channel history command');
  test('sendChannelMessage executes bc channel send command');

  // Cost
  test('getCostSummary returns parsed CostSummary');
  test('getCostSummary returns empty summary on error');

  // Demons
  test('getDemons returns array of demons');
  test('getDemons returns empty array on error');
  test('getDemon returns single demon or null');
  test('getDemonLogs with optional tail parameter');
  test('enableDemon executes enable command');
  test('disableDemon executes disable command');
  test('runDemon executes run command');

  // Processes & Teams (5+ tests)
  test('getProcesses returns process list');
  test('getProcessLogs with optional lines parameter');
  test('getTeams returns teams response');
  test('addTeamMember executes add command');
  test('removeTeamMember executes remove command');
});
```

**Key Test Patterns:**
- Use mock bc service from Subtask 1
- Test graceful fallbacks (empty responses)
- Verify argument construction

---

## Part 2: JSON Schema Validation Tests

### 2.1 File: `src/__tests__/cli/json-schemas.test.ts` (100+ lines, 15+ tests)

**Purpose:** Validate that actual CLI output matches expected TypeScript types

**Implementation Steps:**

1. **Create JSON Schema Validators**
   ```typescript
   // Helper to validate JSON against schema
   function validateSchema<T>(data: unknown, schema: Schema): ValidationResult {
     // Use a JSON schema validation library or custom validator
     // Return { valid: boolean, errors: string[] }
   }
   ```

2. **Test Each Command's JSON Output**
   ```typescript
   describe('CLI JSON Output Schemas', () => {
     test('bc status --json matches StatusResponse schema');
     test('bc channel list --json matches ChannelsResponse schema');
     test('bc channel history <name> --json matches ChannelHistory schema');
     test('bc cost show --json matches CostSummary schema');
     test('bc demon list --json matches Demon[] schema');
     test('bc process list --json matches ProcessListResponse schema');
     test('bc team list --json matches TeamsResponse schema');
     // ... more commands
   });
   ```

3. **Expected JSON Schema for Each Type**
   - StatusResponse: has workspace, total, active, working, agents[]
   - ChannelsResponse: has channels[] with name, members, description
   - CostSummary: has total_cost, by_agent, by_team, by_model
   - Demon: has name, schedule, enabled, created_at, last_run, run_count
   - Process: has name, command, pid, running, started_at
   - Team: has name, members, created_at, updated_at

---

## Part 3: Error Scenario Tests

### 3.1 File: `src/__tests__/cli/error-handling.test.ts` (75+ lines, 10+ tests)

**Tests Required:**
```typescript
describe('CLI Error Scenarios', () => {
  test('invalid command returns error');
  test('missing required arguments returns error');
  test('invalid channel name returns error');
  test('timeout error includes helpful message');
  test('JSON parse error shows output preview');
  test('stderr is included in error message');
  test('environment variable overrides work (BC_BIN, BC_ROOT)');
  test('command with special characters in args');
  test('command with very long output');
  test('concurrent command execution handles errors');
});
```

**Key Test Patterns:**
- Test with invalid command names
- Test with missing required arguments
- Test with malformed JSON responses
- Test timeout scenarios (mock delay)
- Test environment variable handling

---

## Part 4: Performance Benchmarking Tests

### 4.1 File: `src/__tests__/cli/performance.test.ts` (50+ lines, 8+ tests)

**Objective:** Ensure command execution meets performance SLAs

**Test Structure:**
```typescript
describe('CLI Performance Benchmarks', () => {
  const PERFORMANCE_THRESHOLDS = {
    status: 1000,      // ms
    channel: 1000,
    cost: 1000,
    demon: 500,
    process: 500,
    team: 500,
  };

  test('bc status completes in <1s');
  test('bc channel list completes in <1s');
  test('bc cost show completes in <1s');
  test('bc demon list completes in <500ms');
  test('bc process list completes in <500ms');
  test('bc team list completes in <500ms');
  test('concurrent commands total <5s');
  test('performance metrics logged for CI');
});
```

**Implementation:**
- Use high-resolution timer (Date.now() or performance.now())
- Mock bc service to return quickly
- Log performance metrics for tracking
- Fail if threshold exceeded

---

## Part 5: CLI Integration Test Utilities

### 5.1 Required Test Utilities (from Subtask 1)

These utilities will be created by vibrant-cheetah in Subtask 1:

```typescript
// From src/__tests__/utils/testUtils.tsx
export function mockBcService(responses: Record<string, any>)
export function createMockAgent(overrides?: Partial<Agent>): Agent
export function createMockChannel(overrides?: Partial<Channel>): Channel
// ... more utilities
```

### 5.2 Additional CLI Test Utilities to Create

```typescript
// File: src/__tests__/utils/cliTestUtils.ts

export interface MockBcCommand {
  args: string[];
  stdout: string;
  stderr?: string;
  exitCode?: number;
  delay?: number;
}

export class MockBcProcess {
  /**
   * Setup mock responses for specific commands
   * Example: setupResponse(['status'], { ... })
   */
  setupResponse(args: string[], data: any): void

  /**
   * Setup command to fail with exit code
   */
  setupError(args: string[], message: string): void

  /**
   * Setup command to timeout
   */
  setupTimeout(args: string[]): void

  /**
   * Get all commands that were executed
   */
  getExecutedCommands(): string[][]

  /**
   * Verify specific command was called
   */
  expectCommand(args: string[]): void
}

export function captureCommand(
  fn: () => Promise<void>
): { command: string; args: string[] }
```

---

## Part 6: Test Data & Fixtures for CLI Testing

### 6.1 Fixture Files (created in Subtask 1, used in Subtask 2)

**Fixture Structure:**
```
src/__tests__/fixtures/
├── agents.ts          # Agent factory with various states
├── channels.ts        # Channel factory with messages
├── costs.ts           # Cost data factory
├── demons.ts          # Demon/scheduled tasks factory
├── processes.ts       # Process list factory
└── teams.ts           # Team factory
```

**Example Usage in CLI Tests:**
```typescript
test('bc status returns multiple agents', () => {
  const agents = [
    createMockAgent({ name: 'agent-1', state: 'idle' }),
    createMockAgent({ name: 'agent-2', state: 'working' }),
  ];
  mockBc.setupResponse(['status'], { agents });

  const result = await getStatus();
  expect(result.agents).toHaveLength(2);
});
```

---

## Part 7: Integration with Hook Testing (Subtask 2 - Parallel)

While testing CLI service layer, hooks will also be tested:

**Hook Test Structure:**
```
src/__tests__/unit/hooks/
├── useChannels.test.ts     (20+ tests)
├── useAgents.test.ts       (20+ tests)
├── useDashboard.test.ts    (20+ tests)
├── usePolling.test.ts      (15+ tests)
└── ... (8+ more hook files)
```

**Coordination with young-narwhal:**
- CLI tests mock the bc service
- Hook tests use the mocked bc service
- Ensure consistency between both test layers

---

## Implementation Timeline (Subtask 2 Execution)

### Week 2, Monday-Tuesday: CLI Service & JSON Schema Tests
- Create `src/__tests__/unit/services/bc.test.ts` (30+ tests)
- Create `src/__tests__/cli/json-schemas.test.ts` (15+ tests)
- Create `src/__tests__/utils/cliTestUtils.ts` (helper utilities)
- Expected: 45+ tests, ~300 lines

### Week 2, Wednesday: Error & Performance Tests
- Create `src/__tests__/cli/error-handling.test.ts` (10+ tests)
- Create `src/__tests__/cli/performance.test.ts` (8+ tests)
- Expected: 18+ tests, ~125 lines

### Week 2, Thursday: Integration & Validation
- Run full test suite: `cd tui && bun test`
- Verify coverage: Target 95%+ for bc.ts service
- Address any failures from hook integration
- Expected coverage improvement: 11.7% → 40-45%

### Week 2, Friday: PR Review & Finalization
- Code review by vibrant-cheetah and noble-vulture
- Address feedback, iterate
- Merge PR for Subtask 2
- Prepare for Subtask 3 (Component testing)

---

## Success Criteria for Subtask 2

### Code Quality
- [ ] All 150+ tests pass locally and in CI
- [ ] TypeScript compilation succeeds with 0 errors
- [ ] Zero lint warnings/errors
- [ ] Test execution time <15 seconds

### Coverage
- [ ] bc.ts service: 100% coverage
- [ ] All hooks: 95%+ coverage
- [ ] Fixtures: All factories tested
- [ ] Overall coverage improvement: 11.7% → 40-45%

### Functionality
- [ ] All CLI commands tested with success/failure scenarios
- [ ] All JSON schemas validated
- [ ] All error scenarios covered
- [ ] Performance benchmarks established

### Documentation
- [ ] Test utilities documented with examples
- [ ] Fixtures usage documented
- [ ] Performance thresholds documented
- [ ] Error message patterns documented

---

## Dependencies on Subtask 1 Deliverables

These must be complete before starting Subtask 2:

1. ✅ **Test Utilities** (`src/__tests__/utils/testUtils.tsx`)
   - renderWithProviders()
   - mockBcService()
   - createMockAgent(), createMockChannel(), etc.
   - waitForElement() helpers

2. ✅ **Fixture Factories** (`src/__tests__/fixtures/*`)
   - agents.ts, channels.ts, costs.ts, demons.ts, processes.ts, teams.ts
   - Each with realistic data generation

3. ✅ **Setup Configuration** (`src/__tests__/setup.ts`)
   - Global test configuration
   - Mock timers if needed
   - Provider setup/teardown

4. ✅ **Bun Compatibility Fixes**
   - No jest.fn() references
   - Bun-native mocking patterns documented

---

## Blockers & Mitigations

| Blocker | Impact | Mitigation |
|---------|--------|-----------|
| Subtask 1 delayed | Blocks Subtask 2 start | Monitor daily, escalate if >1 day delay |
| BC CLI not available in CI | Can't run real commands | Use mock bc service throughout |
| Fixture data generation slow | Slows tests | Use lazy generation, cache fixtures |
| Codecov integration fails | Can't track coverage | Fallback to local Bun coverage reporting |

---

## Handoff to Subtask 3

Upon completion of Subtask 2:
- PR ready for review with 150+ CLI/hook tests
- Coverage at 40-45% (20+ point improvement)
- All CLI commands validated and benchmarked
- Performance baselines established

**Next Phase (Subtask 3):**
- Component tests (300+ tests)
- View tests (200+ tests)
- Integration tests (70+ tests)
- CI/CD gates (coverage >80% enforcement)
- Target: 80%+ coverage by EOW2

---

## References

**Related Files:**
- `tui/src/services/bc.ts` - Service layer to test (325 lines)
- `tui/src/types/index.ts` - Types to validate against (176 lines)
- `tui/src/hooks/*.ts` - Hooks that use bc service
- `docs/PHASE-2-TESTING-DESIGN.md` - Overall Phase 2 design

**External Resources:**
- Bun Testing: https://bun.sh/docs/test/introduction
- Node child_process: https://nodejs.org/docs/latest/api/child_process.html
- JSON Schema Validation: https://json-schema.org/

---

**Document Status:**
- ✅ Draft Complete: 2026-02-15
- ⏳ Ready for Subtask 1 Completion
- 📊 Target: 40-45% coverage from Subtask 2
- ⏱️ Expected Start: Week 2, Monday

