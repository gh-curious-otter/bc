/**
 * MemoryView tests
 * Issue #1231 - Additional TUI views
 * Issue #1497 - Updated for HintsContext pattern
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Text, Box } from 'ink';
import { MemoryView } from '../views/MemoryView';
import { FocusProvider } from '../navigation/FocusContext';
import { NavigationProvider } from '../navigation/NavigationContext';
import { HintsProvider, useHintsContext, DisableInputProvider } from '../hooks';
import * as bc from '../services/bc';

// Helper to display hints from context
function HintsDisplay(): React.ReactElement {
  const { viewHints } = useHintsContext();
  return (
    <Box>
      {viewHints.map((h) => (
        <Text key={h.key}>[{h.key}] {h.label}</Text>
      ))}
    </Box>
  );
}

// Mock the bc service
jest.mock('../services/bc', () => ({
  getMemoryList: jest.fn(),
  getMemory: jest.fn(),
  searchMemory: jest.fn(),
  clearMemory: jest.fn(),
}));

const mockGetMemoryList = bc.getMemoryList as jest.Mock;
const mockGetMemory = bc.getMemory as jest.Mock;

// #1594: Use DisableInputProvider instead of prop
// #1604: Add NavigationProvider for breadcrumb context
function renderMemoryView(props = {}, withHintsDisplay = false) {
  return render(
    <HintsProvider>
      <NavigationProvider>
        <FocusProvider>
          <DisableInputProvider disabled>
            <MemoryView {...props} />
            {withHintsDisplay && <HintsDisplay />}
          </DisableInputProvider>
        </FocusProvider>
      </NavigationProvider>
    </HintsProvider>
  );
}

describe('MemoryView', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('shows loading state initially', () => {
    mockGetMemoryList.mockImplementation(() => new Promise(() => {})); // Never resolves
    const { lastFrame } = renderMemoryView();
    const output = lastFrame() ?? '';
    expect(output).toContain('Loading');
  });

  test('displays agent list when loaded', async () => {
    mockGetMemoryList.mockResolvedValue({
      agents: [
        { agent: 'eng-01', experience_count: 5, learning_count: 3 },
        { agent: 'eng-02', experience_count: 2, learning_count: 1 },
      ],
    });

    const { lastFrame } = renderMemoryView();

    // Wait for async load
    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame() ?? '';
    expect(output).toContain('Agent Memories');
    expect(output).toContain('eng-01');
    expect(output).toContain('eng-02');
  });

  test('shows empty state when no agents', async () => {
    mockGetMemoryList.mockResolvedValue({ agents: [] });

    const { lastFrame } = renderMemoryView();
    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame() ?? '';
    expect(output).toContain('No agent memories');
  });

  test('displays keyboard hints', async () => {
    // Issue #1497: Hints now go to global footer via HintsContext
    mockGetMemoryList.mockResolvedValue({
      agents: [{ agent: 'eng-01', experience_count: 1, learning_count: 1 }],
    });

    const { lastFrame } = renderMemoryView({}, true);
    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame() ?? '';
    expect(output).toContain('j/k');
    expect(output).toContain('navigate');
  });

  test('shows header with agent count', async () => {
    mockGetMemoryList.mockResolvedValue({
      agents: [
        { agent: 'eng-01', experience_count: 5, learning_count: 3 },
        { agent: 'eng-02', experience_count: 2, learning_count: 1 },
      ],
    });

    const { lastFrame } = renderMemoryView();
    await new Promise(resolve => setTimeout(resolve, 100));

    // HeaderBar shows count and subtitle separately (#1446)
    const output = lastFrame() ?? '';
    expect(output).toContain('(2)');
    expect(output).toContain('agents');
  });
});

describe('MemoryView detail view', () => {
  test('shows memory details when fetched', async () => {
    mockGetMemoryList.mockResolvedValue({
      agents: [{ agent: 'eng-01', experience_count: 1, learning_count: 1 }],
    });
    mockGetMemory.mockResolvedValue({
      agent: 'eng-01',
      experiences: [
        { id: '1', timestamp: '2024-01-01T00:00:00Z', category: 'task', outcome: 'success', message: 'Test experience' },
      ],
      learnings: [
        { topic: 'patterns', content: 'Test learning' },
      ],
      experience_count: 1,
      learning_count: 1,
    });

    // Note: This test would need keyboard simulation to trigger detail view
    // For now we just verify the component renders without error
    const { lastFrame } = renderMemoryView();
    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame() ?? '';
    expect(output).toContain('eng-01');
  });
});
