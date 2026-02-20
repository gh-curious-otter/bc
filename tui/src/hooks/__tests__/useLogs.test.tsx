/**
 * Tests for useLogs hook - Event log fetching and filtering
 * Validates severity helpers and type exports
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing requires DOM environment which is not available in Bun/Ink.
 * These tests focus on utility functions that can be tested without hooks.
 */

import { describe, it, expect } from 'bun:test';
import { getSeverityColor, getSeverityIcon } from '../useLogs';
import type { UseLogsOptions, UseLogsResult, LogSeverity } from '../useLogs';

describe('useLogs - Severity Color Mapping', () => {
  describe('getSeverityColor', () => {
    it('returns red for error types', () => {
      expect(getSeverityColor('error')).toBe('red');
      expect(getSeverityColor('AGENT_ERROR')).toBe('red');
      expect(getSeverityColor('command_failed')).toBe('red');
      expect(getSeverityColor('task_error')).toBe('red');
      expect(getSeverityColor('FAIL')).toBe('red');
    });

    it('returns yellow for warning types', () => {
      expect(getSeverityColor('warning')).toBe('yellow');
      expect(getSeverityColor('AGENT_STUCK')).toBe('yellow');
      expect(getSeverityColor('warn')).toBe('yellow');
      expect(getSeverityColor('STUCK_DETECTED')).toBe('yellow');
    });

    it('returns gray for info types', () => {
      expect(getSeverityColor('agent_started')).toBe('gray');
      expect(getSeverityColor('message_sent')).toBe('gray');
      expect(getSeverityColor('info')).toBe('gray');
      expect(getSeverityColor('agent_stopped')).toBe('gray');
      expect(getSeverityColor('task_completed')).toBe('gray');
    });

    it('handles mixed case event types', () => {
      expect(getSeverityColor('Error')).toBe('red');
      expect(getSeverityColor('ERROR')).toBe('red');
      expect(getSeverityColor('Warning')).toBe('yellow');
      expect(getSeverityColor('WARNING')).toBe('yellow');
      expect(getSeverityColor('Info')).toBe('gray');
      expect(getSeverityColor('INFO')).toBe('gray');
    });

    it('handles event types with prefixes', () => {
      expect(getSeverityColor('AGENT_ERROR_TIMEOUT')).toBe('red');
      expect(getSeverityColor('task_failed_validation')).toBe('red');
      expect(getSeverityColor('warning_threshold_exceeded')).toBe('yellow');
    });

    it('defaults to gray for unknown types', () => {
      expect(getSeverityColor('custom_event')).toBe('gray');
      expect(getSeverityColor('unknown')).toBe('gray');
      expect(getSeverityColor('')).toBe('gray');
    });
  });
});

describe('useLogs - Type Exports', () => {
  it('exports LogSeverity type', () => {
    // Type checking - these should compile without errors
    const info: LogSeverity = 'info';
    const warn: LogSeverity = 'warn';
    const error: LogSeverity = 'error';

    expect(info).toBe('info');
    expect(warn).toBe('warn');
    expect(error).toBe('error');
  });

  it('exports UseLogsOptions interface', () => {
    // Verify the interface shape
    const options: UseLogsOptions = {
      pollInterval: 5000,
      autoPoll: true,
      tail: 50,
      agent: 'eng-01',
      eventType: 'agent_started',
      severity: 'error',
    };

    expect(options.pollInterval).toBe(5000);
    expect(options.autoPoll).toBe(true);
    expect(options.tail).toBe(50);
    expect(options.agent).toBe('eng-01');
    expect(options.eventType).toBe('agent_started');
    expect(options.severity).toBe('error');
  });

  it('allows partial UseLogsOptions', () => {
    const minimalOptions: UseLogsOptions = {};
    const withPoll: UseLogsOptions = { autoPoll: false };
    const withTail: UseLogsOptions = { tail: 100 };

    expect(minimalOptions).toBeDefined();
    expect(withPoll.autoPoll).toBe(false);
    expect(withTail.tail).toBe(100);
  });
});

describe('useLogs - Severity Logic', () => {
  // Test the severity detection patterns

  it('error detection includes fail variants', () => {
    const errorPatterns = ['error', 'fail', 'failed', 'failure', 'ERROR', 'FAIL'];
    for (const pattern of errorPatterns) {
      expect(getSeverityColor(pattern)).toBe('red');
    }
  });

  it('warning detection includes stuck variants', () => {
    const warnPatterns = ['warn', 'warning', 'stuck', 'WARN', 'STUCK'];
    for (const pattern of warnPatterns) {
      expect(getSeverityColor(pattern)).toBe('yellow');
    }
  });

  it('info detection is the fallback', () => {
    // Any event type that doesn't match error or warn patterns
    const infoPatterns = ['started', 'stopped', 'completed', 'sent', 'received'];
    for (const pattern of infoPatterns) {
      expect(getSeverityColor(pattern)).toBe('gray');
    }
  });
});

describe('useLogs - Edge Cases', () => {
  it('handles empty string', () => {
    expect(getSeverityColor('')).toBe('gray');
  });

  it('handles special characters in event type', () => {
    expect(getSeverityColor('error:timeout')).toBe('red');
    expect(getSeverityColor('warn-deprecated')).toBe('yellow');
    expect(getSeverityColor('info_event')).toBe('gray');
  });

  it('handles long event type strings', () => {
    const longError = 'this_is_a_very_long_error_event_type_name';
    const longWarn = 'this_is_a_warning_with_lots_of_details';
    const longInfo = 'some_detailed_information_event_type';

    expect(getSeverityColor(longError)).toBe('red');
    expect(getSeverityColor(longWarn)).toBe('yellow');
    expect(getSeverityColor(longInfo)).toBe('gray');
  });
});

describe('useLogs - Severity Icon Mapping (#1220)', () => {
  describe('getSeverityIcon', () => {
    it('returns ✗ for error types', () => {
      expect(getSeverityIcon('error')).toBe('✗');
      expect(getSeverityIcon('AGENT_ERROR')).toBe('✗');
      expect(getSeverityIcon('FAIL')).toBe('✗');
    });

    it('returns ⚠ for warning types', () => {
      expect(getSeverityIcon('warning')).toBe('⚠');
      expect(getSeverityIcon('AGENT_STUCK')).toBe('⚠');
      expect(getSeverityIcon('warn')).toBe('⚠');
    });

    it('returns · for info types', () => {
      expect(getSeverityIcon('agent_started')).toBe('·');
      expect(getSeverityIcon('message_sent')).toBe('·');
      expect(getSeverityIcon('info')).toBe('·');
    });

    it('handles case insensitivity', () => {
      expect(getSeverityIcon('Error')).toBe('✗');
      expect(getSeverityIcon('WARNING')).toBe('⚠');
      expect(getSeverityIcon('Info')).toBe('·');
    });
  });
});
