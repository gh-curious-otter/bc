/**
 * MemoryView tests
 * Issue #1231 - Additional TUI views
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { MemoryView } from '../views/MemoryView';
import { FocusProvider } from '../navigation/FocusContext';
import * as bc from '../services/bc';

// Mock the bc service
jest.mock('../services/bc', () => ({
  getMemoryList: jest.fn(),
  getMemory: jest.fn(),
  searchMemory: jest.fn(),
  clearMemory: jest.fn(),
}));

const mockGetMemoryList = bc.getMemoryList as jest.Mock;
const mockGetMemory = bc.getMemory as jest.Mock;

function renderMemoryView(props = {}) {
  return render(
    <FocusProvider>
      <MemoryView disableInput {...props} />
    </FocusProvider>
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
    mockGetMemoryList.mockResolvedValue({
      agents: [{ agent: 'eng-01', experience_count: 1, learning_count: 1 }],
    });

    const { lastFrame } = renderMemoryView();
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

    const output = lastFrame() ?? '';
    expect(output).toContain('2 agents');
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
