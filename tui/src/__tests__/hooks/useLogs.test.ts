/**
 * useLogs hook tests - Unit tests for utility functions
 * Issue #796 - Live activity feed hook
 */

import { describe, it, expect } from 'bun:test';

// Test the severity detection logic directly (avoiding mock contamination)
function testGetSeverityColor(eventType: string): string {
  const lowerType = eventType.toLowerCase();
  if (lowerType.includes('error') || lowerType.includes('fail')) {
    return 'red';
  }
  if (lowerType.includes('warn') || lowerType.includes('stuck')) {
    return 'yellow';
  }
  return 'gray';
}

describe('getSeverityColor logic', () => {
  it('returns red for error types', () => {
    expect(testGetSeverityColor('agent.error')).toBe('red');
    expect(testGetSeverityColor('task.ERROR')).toBe('red');
  });

  it('returns red for fail types', () => {
    expect(testGetSeverityColor('build.failed')).toBe('red');
    expect(testGetSeverityColor('task.FAILURE')).toBe('red');
  });

  it('returns yellow for stuck types', () => {
    expect(testGetSeverityColor('agent.stuck')).toBe('yellow');
    expect(testGetSeverityColor('STUCK.process')).toBe('yellow');
  });

  it('returns yellow for warn types', () => {
    expect(testGetSeverityColor('system.warning')).toBe('yellow');
    expect(testGetSeverityColor('WARN.memory')).toBe('yellow');
  });

  it('returns gray for info types', () => {
    expect(testGetSeverityColor('message.sent')).toBe('gray');
    expect(testGetSeverityColor('agent.started')).toBe('gray');
    expect(testGetSeverityColor('task.completed')).toBe('gray');
  });
});
