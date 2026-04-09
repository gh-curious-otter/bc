/**
 * Manual mock for useLogs hook.
 * Bun auto-discovers __mocks__ directories — no vi.mock() factory needed,
 * which avoids the "unhandled error between tests" Bun quirk.
 */

import { vi } from 'bun:test';

const defaultData = [
  { ts: '2026-02-16T10:00:00Z', type: 'message.sent', agent: 'eng-01', message: 'Working on task' },
  { ts: '2026-02-16T10:01:00Z', type: 'agent.error', agent: 'eng-02', message: 'Build failed' },
  { ts: '2026-02-16T10:02:00Z', type: 'agent.stuck', agent: 'eng-03', message: 'Waiting for response' },
];

export const useLogs = vi.fn(() => ({
  data: defaultData,
  loading: false,
  error: null,
  severityFilter: null,
  filterBySeverity: vi.fn(),
  refresh: vi.fn(),
}));

export function getSeverityColor(type: string): string {
  const lower = type.toLowerCase();
  if (lower.includes('error') || lower.includes('fail')) return 'red';
  if (lower.includes('warn') || lower.includes('stuck')) return 'yellow';
  return 'gray';
}
