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
      expect(resolveCommand('?')).toBe('help');
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
      expect(results.some((r) => r.command.command === 'dashboard')).toBe(true);
    });

    test('finds commands by alias', () => {
      const results = searchCommands('ag');
      expect(results.some((r) => r.command.command === 'agents')).toBe(true);
    });

    test('includes action commands in search results (#1836)', () => {
      const results = searchCommands('q');
      expect(results.some((r) => r.command.command === 'quit')).toBe(true);
    });

    test('ranks exact matches higher', () => {
      const results = searchCommands('help');
      expect(results[0].command.command).toBe('help');
    });
  });

  describe('searchCommands LRU (#1871)', () => {
    test('shows recent commands first when query is empty', () => {
      const results = searchCommands('', ['logs', 'costs']);
      expect(results[0].command.command).toBe('logs');
      expect(results[1].command.command).toBe('costs');
      expect(results[0].command.section).toBe('RECENT');
      expect(results[1].command.section).toBe('RECENT');
    });

    test('recent commands have higher score than non-recent when empty', () => {
      const results = searchCommands('', ['costs']);
      const recent = results.find((r) => r.command.command === 'costs');
      const nonRecent = results.find((r) => r.command.command === 'dashboard');
      expect(recent).toBeDefined();
      expect(nonRecent).toBeDefined();
      expect(recent!.score).toBeGreaterThan(nonRecent!.score);
    });

    test('non-recent commands still appear after recent ones', () => {
      const results = searchCommands('', ['logs']);
      // First is recent
      expect(results[0].command.command).toBe('logs');
      // Rest are all non-recent view commands
      const rest = results.slice(1);
      expect(rest.length).toBeGreaterThan(0);
      expect(rest.every((r) => r.command.section !== 'RECENT')).toBe(true);
    });

    test('LRU boost gives tiebreak advantage when query has text', () => {
      // Both 'logs' and 'roles' contain 'lo' — with LRU boost, 'logs' should rank first
      const withLru = searchCommands('lo', ['logs']);
      const logsIdx = withLru.findIndex((r) => r.command.command === 'logs');
      expect(logsIdx).toBe(0);
    });

    test('works without LRU parameter (backward compatible)', () => {
      const results = searchCommands('dash');
      expect(results.some((r) => r.command.command === 'dashboard')).toBe(true);
    });

    test('ignores invalid LRU entries', () => {
      const results = searchCommands('', ['nonexistent', 'logs']);
      // nonexistent is silently skipped, logs still shows as recent
      expect(results[0].command.command).toBe('logs');
      expect(results[0].command.section).toBe('RECENT');
    });
  });
});
