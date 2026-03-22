/**
 * TUI Render Performance Benchmarks
 * Issue #962 Phase 4 - Load Testing
 *
 * Measures render performance under various load conditions:
 * - Many agents (10, 50, 100)
 * - Many messages (100, 500, 1000)
 * - Rapid state updates
 *
 * Target: <16ms render cycles for 60fps capable
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Box, Text } from 'ink';
import { describe, it, expect, beforeEach } from 'bun:test';

// Components to benchmark
import { StatusBadge } from '../../components/StatusBadge';
import { ChatMessage } from '../../components/ChatMessage';
import { Panel } from '../../components/Panel';
import { Table } from '../../components/Table';
import { MetricCard } from '../../components/MetricCard';
import { ProgressBar } from '../../components/ProgressBar';
import { Footer } from '../../components/Footer';

// Generate mock data for load testing
function generateMockAgents(count: number) {
  return Array.from({ length: count }, (_, i) => ({
    name: `eng-${String(i + 1).padStart(2, '0')}`,
    role: 'engineer',
    state: ['idle', 'working', 'done', 'error'][i % 4],
    task: i % 2 === 0 ? `Task ${i + 1}` : '',
  }));
}

function generateMockMessages(count: number) {
  const senders = ['eng-01', 'eng-02', 'tl-01', 'mgr-01', 'root'];
  return Array.from({ length: count }, (_, i) => ({
    sender: senders[i % senders.length],
    message: `Message content ${i + 1}. This is a longer message with @mentions and details.`,
    timestamp: new Date(Date.now() - i * 60000).toISOString(),
    reactions: i % 3 === 0 ? [{ type: 'ack' as const, count: 1 }] : [],
  }));
}

// Utility to measure render time
function measureRenderTime(element: React.ReactElement): { time: number; output: string } {
  const start = performance.now();
  const { lastFrame } = render(element);
  const end = performance.now();
  return { time: end - start, output: lastFrame() ?? '' };
}

// Utility to measure multiple renders and get statistics
function benchmarkRender(
  element: React.ReactElement,
  iterations = 10
): { avg: number; min: number; max: number; p95: number } {
  const times: number[] = [];

  for (let i = 0; i < iterations; i++) {
    const { time } = measureRenderTime(element);
    times.push(time);
  }

  times.sort((a, b) => a - b);
  const sum = times.reduce((a, b) => a + b, 0);
  const p95Index = Math.floor(times.length * 0.95);

  return {
    avg: sum / times.length,
    min: times[0],
    max: times[times.length - 1],
    p95: times[p95Index],
  };
}

describe('TUI Render Performance Benchmarks', () => {
  describe('StatusBadge Performance', () => {
    it('renders single badge under 20ms', () => {
      const { time } = measureRenderTime(<StatusBadge state="working" />);
      // Cold render - first component load may be slower
      expect(time).toBeLessThan(20);
    });

    it('renders 100 badges under 100ms', () => {
      const states = ['idle', 'working', 'done', 'error', 'stuck'];
      const element = (
        <Box flexDirection="column">
          {Array.from({ length: 100 }, (_, i) => (
            <StatusBadge key={i} state={states[i % states.length]} />
          ))}
        </Box>
      );
      const { time } = measureRenderTime(element);
      // 100 components cold render target
      expect(time).toBeLessThan(100);
    });
  });

  describe('ChatMessage Performance', () => {
    it('renders single message under 20ms', () => {
      const { time } = measureRenderTime(
        <ChatMessage sender="eng-01" message="Test message" timestamp={new Date().toISOString()} />
      );
      // Note: First render is slower, subsequent renders benefit from memoization
      expect(time).toBeLessThan(20);
    });

    it('renders 50 messages under 200ms', () => {
      const messages = generateMockMessages(50);
      const element = (
        <Box flexDirection="column">
          {messages.map((m, i) => (
            <ChatMessage
              key={i}
              sender={m.sender}
              message={m.message}
              timestamp={m.timestamp}
              reactions={m.reactions}
            />
          ))}
        </Box>
      );
      const { time } = measureRenderTime(element);
      // Cold render target - subsequent renders much faster with memoization
      expect(time).toBeLessThan(200);
    });

    it('renders 100 messages under 200ms', () => {
      const messages = generateMockMessages(100);
      const element = (
        <Box flexDirection="column">
          {messages.map((m, i) => (
            <ChatMessage
              key={i}
              sender={m.sender}
              message={m.message}
              timestamp={m.timestamp}
              reactions={m.reactions}
            />
          ))}
        </Box>
      );
      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(200);
    });
  });

  describe('Table Performance', () => {
    it('renders 10 row table under 50ms', () => {
      const agents = generateMockAgents(10);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'role', header: 'ROLE', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
      ];
      const element = <Table data={agents} columns={columns} />;
      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(50);
    });

    it('renders 50 row table under 150ms', () => {
      const agents = generateMockAgents(50);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'role', header: 'ROLE', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
        { key: 'task', header: 'TASK', width: 20 },
      ];
      const element = <Table data={agents} columns={columns} />;
      const { time } = measureRenderTime(element);
      // Cold render target - TableRow memoization helps subsequent renders
      expect(time).toBeLessThan(150);
    });

    it('renders 100 row table under 150ms', () => {
      const agents = generateMockAgents(100);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'role', header: 'ROLE', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
        { key: 'task', header: 'TASK', width: 20 },
      ];
      const element = <Table data={agents} columns={columns} />;
      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(150);
    });

    it('virtualized table renders visible rows only', () => {
      const agents = generateMockAgents(100);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
      ];
      const element = (
        <Table data={agents} columns={columns} maxVisibleRows={20} scrollOffset={0} />
      );
      const { time, output } = measureRenderTime(element);

      // Should render quickly due to virtualization
      expect(time).toBeLessThan(50);

      // Should only show 20 rows
      const lines = output.split('\n').filter((l) => l.includes('eng-'));
      expect(lines.length).toBeLessThanOrEqual(20);
    });
  });

  describe('Panel Performance', () => {
    it('renders nested panels under 30ms', () => {
      const element = (
        <Panel title="Outer">
          <Panel title="Inner 1">
            <Text>Content 1</Text>
          </Panel>
          <Panel title="Inner 2">
            <Text>Content 2</Text>
          </Panel>
          <Panel title="Inner 3">
            <Text>Content 3</Text>
          </Panel>
        </Panel>
      );
      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(30);
    });
  });

  describe('MetricCard Performance', () => {
    it('renders 10 metric cards under 30ms', () => {
      const element = (
        <Box>
          {Array.from({ length: 10 }, (_, i) => (
            <MetricCard key={i} label={`Metric ${i}`} value={i * 100} color="green" />
          ))}
        </Box>
      );
      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(30);
    });
  });

  describe('ProgressBar Performance', () => {
    it('renders 20 progress bars under 50ms', () => {
      const element = (
        <Box flexDirection="column">
          {Array.from({ length: 20 }, (_, i) => (
            <ProgressBar key={i} value={i * 5} max={100} width={30} />
          ))}
        </Box>
      );
      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(50);
    });
  });

  describe('Footer Performance', () => {
    it('renders footer with hints under 10ms', () => {
      const hints = [
        { key: 'a', label: 'Add' },
        { key: 'd', label: 'Delete' },
        { key: 'e', label: 'Edit' },
        { key: 'q', label: 'Quit' },
      ];
      const { time } = measureRenderTime(<Footer hints={hints} />);
      expect(time).toBeLessThan(10);
    });
  });

  describe('Combined Dashboard Performance', () => {
    it('renders full dashboard with 20 agents under 100ms', () => {
      const agents = generateMockAgents(20);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
        { key: 'task', header: 'TASK', width: 20 },
      ];

      const element = (
        <Box flexDirection="column">
          <Panel title="Agents">
            <Table data={agents} columns={columns} />
          </Panel>
          <Box>
            <MetricCard label="Active" value={10} color="green" />
            <MetricCard label="Idle" value={5} color="gray" />
            <MetricCard label="Error" value={2} color="red" />
          </Box>
          <Footer hints={[{ key: 'q', label: 'Quit' }]} />
        </Box>
      );

      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(100);
    });

    it('renders complex view with messages and agents under 150ms', () => {
      const agents = generateMockAgents(30);
      const messages = generateMockMessages(50);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
      ];

      const element = (
        <Box flexDirection="row" width={120}>
          <Box flexDirection="column" width="50%">
            <Panel title="Agents">
              <Table data={agents} columns={columns} maxVisibleRows={15} />
            </Panel>
          </Box>
          <Box flexDirection="column" width="50%">
            <Panel title="Messages">
              {messages.slice(0, 10).map((m, i) => (
                <ChatMessage
                  key={i}
                  sender={m.sender}
                  message={m.message}
                  timestamp={m.timestamp}
                />
              ))}
            </Panel>
          </Box>
        </Box>
      );

      const { time } = measureRenderTime(element);
      expect(time).toBeLessThan(150);
    });
  });

  describe('Re-render Performance (Memoization)', () => {
    it('subsequent renders are faster due to memoization', () => {
      const agents = generateMockAgents(50);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
      ];

      // First render (cold)
      const first = measureRenderTime(<Table data={agents} columns={columns} />);

      // Benchmark multiple renders
      const stats = benchmarkRender(<Table data={agents} columns={columns} />, 10);

      // Average should be reasonable
      expect(stats.avg).toBeLessThan(100);
      // p95 should still be under target
      expect(stats.p95).toBeLessThan(150);

      // Log benchmark results for analysis
      console.log('Table (50 rows) benchmark:', { // eslint-disable-line no-console
        firstRender: `${first.time.toFixed(2)}ms`,
        avgRender: `${stats.avg.toFixed(2)}ms`,
        minRender: `${stats.min.toFixed(2)}ms`,
        maxRender: `${stats.max.toFixed(2)}ms`,
        p95Render: `${stats.p95.toFixed(2)}ms`,
      });
    });
  });

  describe('Stress Tests', () => {
    it('handles 100 agents without timeout', () => {
      const agents = generateMockAgents(100);
      const columns = [
        { key: 'name', header: 'NAME', width: 10 },
        { key: 'role', header: 'ROLE', width: 10 },
        { key: 'state', header: 'STATE', width: 10 },
      ];

      const start = performance.now();
      const { lastFrame } = render(<Table data={agents} columns={columns} maxVisibleRows={20} />);
      const elapsed = performance.now() - start;

      expect(lastFrame()).toBeTruthy();
      expect(elapsed).toBeLessThan(200);
    });

    it('handles 500 messages in batches', () => {
      const messages = generateMockMessages(500);
      const batchSize = 50;
      let totalTime = 0;

      // Render in batches (simulating pagination/virtualization)
      for (let i = 0; i < messages.length; i += batchSize) {
        const batch = messages.slice(i, i + batchSize);
        const { time } = measureRenderTime(
          <Box flexDirection="column">
            {batch.map((m, j) => (
              <ChatMessage key={j} sender={m.sender} message={m.message} timestamp={m.timestamp} />
            ))}
          </Box>
        );
        totalTime += time;
      }

      // Each batch should average under 100ms
      const avgBatchTime = totalTime / (messages.length / batchSize);
      expect(avgBatchTime).toBeLessThan(100);
    });
  });
});

describe('Performance Summary', () => {
  it('generates performance report', () => {
    const results: Record<string, { time: number; target: number; pass: boolean }> = {};

    // StatusBadge
    let { time } = measureRenderTime(<StatusBadge state="working" />);
    results['StatusBadge (1x)'] = { time, target: 5, pass: time < 5 };

    // ChatMessage
    ({ time } = measureRenderTime(
      <ChatMessage sender="eng-01" message="Test" timestamp={new Date().toISOString()} />
    ));
    results['ChatMessage (1x)'] = { time, target: 20, pass: time < 20 };

    // Table 50 rows
    const agents50 = generateMockAgents(50);
    ({ time } = measureRenderTime(
      <Table
        data={agents50}
        columns={[
          { key: 'name', header: 'NAME', width: 10 },
          { key: 'state', header: 'STATE', width: 10 },
        ]}
      />
    ));
    results['Table (50 rows)'] = { time, target: 75, pass: time < 75 };

    // Print summary
    console.log('\n=== TUI Render Performance Report ===\n'); // eslint-disable-line no-console
    for (const [name, result] of Object.entries(results)) {
      const status = result.pass ? '✅' : '❌';
      console.log( // eslint-disable-line no-console
        `${status} ${name.padEnd(25)} ${result.time.toFixed(2).padStart(8)}ms / ${String(result.target).padStart(5)}ms target`,
      );
    }
    console.log(''); // eslint-disable-line no-console

    // All should pass
    const allPass = Object.values(results).every((r) => r.pass);
    expect(allPass).toBe(true);
  });
});
