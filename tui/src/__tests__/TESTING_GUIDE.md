# TUI Testing Guide

## Overview

This guide explains how to write tests for the bc TUI components using the testing framework set up in Phase 2.

## Table of Contents

1. [Setup](#setup)
2. [Component Testing](#component-testing)
3. [Hook Testing](#hook-testing)
4. [Service Testing](#service-testing)
5. [Best Practices](#best-practices)
6. [Common Patterns](#common-patterns)

---

## Setup

### Test Environment

Tests run using Bun test runner with `ink-testing-library` for terminal UI testing.

**Key files:**
- `__tests__/setup.ts` - Global test configuration
- `__tests__/utils/testUtils.tsx` - Common test utilities
- `__tests__/fixtures/index.ts` - Mock data generators
- `__tests__/mocks/bc.ts` - Mock bc service

### Running Tests

```bash
# Run all tests
bun test

# Run tests in watch mode
bun test --watch

# Run specific test file
bun test src/__tests__/components/StatusBadge.test.tsx

# Run with coverage
bun test --coverage
```

---

## Component Testing

### Basic Component Test

```typescript
import { render } from 'ink-testing-library';
import { describe, test, expect } from 'bun:test';
import { StatusBadge } from '../../components/StatusBadge';
import { renderWithProviders } from '../utils/testUtils';

describe('StatusBadge', () => {
  test('renders idle state', () => {
    const { lastFrame } = renderWithProviders(
      <StatusBadge state="idle" />,
      { disableInput: true }
    );
    expect(lastFrame()).toContain('○');
  });

  test('renders working state', () => {
    const { lastFrame } = renderWithProviders(
      <StatusBadge state="working" />,
      { disableInput: true }
    );
    expect(lastFrame()).toContain('◐');
  });
});
```

### Testing with Props

```typescript
test('displays label', () => {
  const { lastFrame } = renderWithProviders(
    <MetricCard value={42} label="Tests" color="green" />,
    { disableInput: true }
  );

  const output = lastFrame();
  expect(output).toContain('42');
  expect(output).toContain('Tests');
});
```

### Testing State Changes

```typescript
test('updates on prop change', () => {
  const { rerender, lastFrame } = renderWithProviders(
    <StatusBadge state="idle" />,
    { disableInput: true }
  );

  expect(lastFrame()).toContain('○');

  rerender(<StatusBadge state="working" />);
  expect(lastFrame()).toContain('◐');
});
```

---

## Hook Testing

### Basic Hook Test

```typescript
import { renderHook, act } from 'ink-testing-library';
import { describe, test, expect } from 'bun:test';
import { useListNavigation } from '../../hooks/useListNavigation';

describe('useListNavigation', () => {
  test('initializes with index 0', () => {
    const { result } = renderHook(() => useListNavigation(5));

    expect(result.current.selectedIndex).toBe(0);
  });

  test('increments index on next', () => {
    const { result } = renderHook(() => useListNavigation(5));

    act(() => {
      result.current.selectNext();
    });

    expect(result.current.selectedIndex).toBe(1);
  });

  test('wraps around at end', () => {
    const { result } = renderHook(() => useListNavigation(3));

    act(() => {
      result.current.selectLast();
    });

    expect(result.current.selectedIndex).toBe(2);

    act(() => {
      result.current.selectNext();
    });

    expect(result.current.selectedIndex).toBe(0);
  });
});
```

### Testing Async Hooks

```typescript
test('fetches data on mount', async () => {
  const { result, waitForNextUpdate } = renderHook(() => useAgents());

  expect(result.current.loading).toBe(true);
  expect(result.current.data).toBe(undefined);

  await waitForNextUpdate();

  expect(result.current.loading).toBe(false);
  expect(result.current.data).toBeDefined();
  expect(result.current.data.length).toBeGreaterThan(0);
});
```

---

## Service Testing

### Mock Service Test

```typescript
import { describe, test, expect } from 'bun:test';
import { createMockBcService } from '../mocks/bc';
import { createMockAgents } from '../fixtures';

describe('MockBcService', () => {
  test('returns mocked agents', async () => {
    const mockAgents = createMockAgents(3);
    const service = createMockBcService({ agents: mockAgents });

    const result = await service.execute('status');

    expect(result.agents.length).toBe(3);
    expect(result.agents[0].name).toBe('agent-1');
  });

  test('tracks command calls', async () => {
    const service = createMockBcService();

    await service.execute('status');
    await service.execute('channel', ['list']);

    service.assertCalled('status');
    service.assertCalled('channel', ['list']);
    expect(service.getCallCount('status')).toBe(1);
  });

  test('can simulate failures', async () => {
    const service = createMockBcService({
      shouldFail: true,
      failureMessage: 'Network error',
    });

    try {
      await service.execute('status');
      throw new Error('Expected error');
    } catch (err: any) {
      expect(err.message).toContain('Network error');
    }
  });
});
```

---

## Best Practices

### 1. Use Fixtures for Mock Data

```typescript
// ✅ GOOD - Use fixture factory
const agents = createMockAgents(3);
const { lastFrame } = renderWithProviders(
  <AgentList agents={agents} />,
  { disableInput: true }
);

// ❌ BAD - Manual data creation
const agents = [
  { id: '1', name: 'agent-1', state: 'idle', /* 20 more fields */ },
  // ...
];
```

### 2. Test Behavior, Not Implementation

```typescript
// ✅ GOOD - Test what user sees
test('displays sorted agents', () => {
  const { lastFrame } = renderWithProviders(
    <AgentList agents={unsortedAgents} />,
    { disableInput: true }
  );
  expect(lastFrame()).toContain('agent-1\nagent-2\nagent-3');
});

// ❌ BAD - Test internal implementation
test('calls sort function', () => {
  const mockSort = jest.fn();
  // ...
});
```

### 3. Isolate Component Dependencies

```typescript
// ✅ GOOD - Mock external dependencies
const mockChannelData = createMockChannels(3);
const { lastFrame } = renderWithProviders(
  <ChannelList channels={mockChannelData} />,
  { disableInput: true }
);

// ❌ BAD - Let component fetch real data
const { lastFrame } = renderWithProviders(
  <ChannelList />, // Tries to fetch real channels
  { disableInput: true }
);
```

### 4. Group Related Tests

```typescript
describe('StatusBadge', () => {
  describe('rendering', () => {
    test('renders idle state', () => { /* ... */ });
    test('renders working state', () => { /* ... */ });
  });

  describe('colors', () => {
    test('uses cyan for idle', () => { /* ... */ });
    test('uses green for working', () => { /* ... */ });
  });
});
```

### 5. Use Descriptive Test Names

```typescript
// ✅ GOOD - Clear what is being tested
test('displays error message when agent not found', () => { /* ... */ });
test('disables send button when no text entered', () => { /* ... */ });

// ❌ BAD - Vague names
test('test error', () => { /* ... */ });
test('button test', () => { /* ... */ });
```

---

## Common Patterns

### Testing Navigation

```typescript
test('navigates between agents', () => {
  const agents = createMockAgents(3);
  const { lastFrame } = renderWithProviders(
    <AgentList agents={agents} />,
    { disableInput: true }
  );

  expect(lastFrame()).toContain('agent-1');

  // Simulate arrow down key
  act(() => {
    hook.selectNext();
  });

  expect(lastFrame()).toContain('agent-2');
});
```

### Testing Data Loading

```typescript
test('shows loading state then data', async () => {
  const { result } = renderHook(() => useAgents());

  // Initially loading
  expect(result.current.loading).toBe(true);

  // Wait for data
  await waitFor(() => !result.current.loading);

  // Data loaded
  expect(result.current.data).toBeDefined();
  expect(result.current.error).toBe(null);
});
```

### Testing Error Handling

```typescript
test('displays error when fetch fails', async () => {
  const service = createMockBcService({
    shouldFail: true,
    failureMessage: 'Failed to load',
  });

  const { result } = renderHook(() => useAgents(service));

  await waitFor(() => result.current.error);

  expect(result.current.error).toContain('Failed to load');
  expect(result.current.data).toBeNull();
});
```

### Testing User Interactions

```typescript
test('sends message on enter key', () => {
  const mockSend = jest.fn();
  const { lastFrame } = renderWithProviders(
    <MessageInput onSend={mockSend} />,
    { disableInput: false }
  );

  // Simulate typing
  act(() => {
    fireEvent('j', 'test message');
    fireEvent(Enter Key);
  });

  expect(mockSend).toHaveBeenCalledWith('test message');
});
```

---

## Debugging Tests

### View Output

```typescript
test('debug output', () => {
  const { lastFrame } = renderWithProviders(
    <AgentList agents={agents} />,
    { disableInput: true }
  );

  // Print the entire rendered output
  console.log(lastFrame());
});
```

### Use Verbose Mode

```bash
bun test --verbose
```

### Check Call History

```typescript
test('verify calls', async () => {
  const service = createMockBcService();

  await service.execute('status');
  console.log(service.getCallHistory());
  // [{ command: 'status', args: [] }]
});
```

---

## Resources

- [Bun Test Runner](https://bun.sh/docs/test/introduction)
- [ink-testing-library](https://github.com/vadimdemedes/ink-testing-library)
- [React Testing Best Practices](https://kentcdodds.com/blog/common-mistakes-with-react-testing-library)

---

## Need Help?

- Check existing tests in `__tests__/` for examples
- Review `testUtils.tsx` for available helpers
- Review `fixtures/index.ts` for mock data factories
- Review `mocks/bc.ts` for service mocking

---

**Happy testing!** 🧪
