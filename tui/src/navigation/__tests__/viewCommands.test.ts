/**
 * viewCommands tests
 * #1836: Vim-style command mode — :q, :q!, action commands
 */

import { describe, test, expect } from 'bun:test';
import { searchCommands, resolveCommand, resolveAction } from '../viewCommands';

describe('viewCommands', () => {
  describe('resolveCommand (view navigation)', () => {
    test('resolves full command names', () => {
      expect(resolveCommand('dashboard')).toBe('dashboard');
      expect(resolveCommand('agents')).toBe('agents');
      expect(resolveCommand('help')).toBe('help');
    });

    test('resolves aliases', () => {
      expect(resolveCommand('dash')).toBe('dashboard');
      expect(resolveCommand('ag')).toBe('agents');
      expect(resolveCommand('mem')).toBe('memory');
      expect(resolveCommand('m')).toBe('memory');
      expect(resolveCommand('?')).toBe('help');
    });

    test('resolves :memory and :mem to memory view', () => {
      expect(resolveCommand('memory')).toBe('memory');
      expect(resolveCommand('mem')).toBe('memory');
      expect(resolveCommand('m')).toBe('memory');
    });

    test('returns null for unknown commands', () => {
      expect(resolveCommand('unknown')).toBeNull();
      expect(resolveCommand('')).toBeNull();
    });

    test('is case-insensitive', () => {
      expect(resolveCommand('Dashboard')).toBe('dashboard');
      expect(resolveCommand('AGENTS')).toBe('agents');
    });
  });

  describe('resolveAction (#1836)', () => {
    test('resolves :q to quit', () => {
      expect(resolveAction('q')).toBe('quit');
    });

    test('resolves :quit to quit', () => {
      expect(resolveAction('quit')).toBe('quit');
    });

    test('resolves :q! to force-quit', () => {
      expect(resolveAction('q!')).toBe('force-quit');
    });

    test('resolves :quit! to force-quit', () => {
      expect(resolveAction('quit!')).toBe('force-quit');
    });

    test('returns null for non-action commands', () => {
      expect(resolveAction('dashboard')).toBeNull();
      expect(resolveAction('agents')).toBeNull();
      expect(resolveAction('')).toBeNull();
    });

    test('does not conflict with view commands', () => {
      // 'q' is an action, not a view
      expect(resolveCommand('q')).toBeNull();
      expect(resolveAction('q')).toBe('quit');
    });
  });

  describe('searchCommands', () => {
    test('returns all commands when query is empty', () => {
      const results = searchCommands('');
      expect(results.length).toBeGreaterThan(0);
    });

    test('finds commands by prefix', () => {
      const results = searchCommands('dash');
      expect(results.some(r => r.command.command === 'dashboard')).toBe(true);
    });

    test('finds commands by alias', () => {
      const results = searchCommands('ag');
      expect(results.some(r => r.command.command === 'agents')).toBe(true);
    });

    test('includes action commands in search results (#1836)', () => {
      const results = searchCommands('q');
      expect(results.some(r => r.command.command === 'quit')).toBe(true);
    });

    test('ranks exact matches higher', () => {
      const results = searchCommands('help');
      expect(results[0].command.command).toBe('help');
    });
  });
});
