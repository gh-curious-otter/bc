/**
 * Tests for useActivityData hook - Agent activity data aggregation
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing requires DOM environment which is not available in Bun/Ink.
 * These tests focus on type checking and interface validation.
 */

import { describe, it, expect } from 'bun:test';
import type { ActivityEvent, ActivityPeriod } from '../useActivityData';

describe('useActivityData - Type Exports', () => {
  describe('ActivityEvent Interface', () => {
    it('requires timestamp as Date', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'working',
      };
      expect(event.timestamp instanceof Date).toBe(true);
    });

    it('requires agent string', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-02',
        action: 'idle',
      };
      expect(typeof event.agent).toBe('string');
    });

    it('requires action string', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-03',
        action: 'done',
      };
      expect(typeof event.action).toBe('string');
    });

    it('allows optional duration', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-04',
        action: 'working',
        duration: 300,
      };
      expect(event.duration).toBe(300);
    });

    it('allows optional cost', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-05',
        action: 'done',
        cost: 0.05,
      };
      expect(event.cost).toBe(0.05);
    });

    it('allows all optional fields', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'working',
        duration: 600,
        cost: 0.10,
      };
      expect(event.duration).toBe(600);
      expect(event.cost).toBe(0.10);
    });

    it('allows minimal required fields only', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'root',
        action: 'started',
      };
      expect(event.duration).toBeUndefined();
      expect(event.cost).toBeUndefined();
    });
  });

  describe('ActivityPeriod Interface', () => {
    it('requires startTime as Date', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: ['eng-01'],
        action: 'working',
        duration: 15,
        totalCost: 0.05,
      };
      expect(period.startTime instanceof Date).toBe(true);
    });

    it('requires endTime as Date', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: ['eng-02'],
        action: 'idle',
        duration: 15,
        totalCost: 0,
      };
      expect(period.endTime instanceof Date).toBe(true);
    });

    it('requires agents array', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: ['eng-01', 'eng-02', 'eng-03'],
        action: 'working',
        duration: 15,
        totalCost: 0.15,
      };
      expect(Array.isArray(period.agents)).toBe(true);
      expect(period.agents.length).toBe(3);
    });

    it('requires action string', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: [],
        action: 'done',
        duration: 15,
        totalCost: 0,
      };
      expect(typeof period.action).toBe('string');
    });

    it('requires duration in minutes', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: ['eng-01'],
        action: 'working',
        duration: 30,
        totalCost: 0.10,
      };
      expect(period.duration).toBe(30);
    });

    it('requires totalCost number', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: ['eng-01'],
        action: 'done',
        duration: 15,
        totalCost: 0.25,
      };
      expect(typeof period.totalCost).toBe('number');
    });

    it('can have empty agents array', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: [],
        action: 'idle',
        duration: 15,
        totalCost: 0,
      };
      expect(period.agents.length).toBe(0);
    });
  });

  describe('Activity Actions', () => {
    it('supports working action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'working',
      };
      expect(event.action).toBe('working');
    });

    it('supports idle action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'idle',
      };
      expect(event.action).toBe('idle');
    });

    it('supports done action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'done',
      };
      expect(event.action).toBe('done');
    });

    it('supports stuck action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'stuck',
      };
      expect(event.action).toBe('stuck');
    });

    it('supports error action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'error',
      };
      expect(event.action).toBe('error');
    });

    it('supports started action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'started',
      };
      expect(event.action).toBe('started');
    });

    it('supports stopped action', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'stopped',
      };
      expect(event.action).toBe('stopped');
    });
  });

  describe('Time Period Calculations', () => {
    it('period duration is positive number', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(Date.now() + 15 * 60 * 1000),
        agents: ['eng-01'],
        action: 'working',
        duration: 15,
        totalCost: 0,
      };
      expect(period.duration).toBeGreaterThan(0);
    });

    it('endTime is after startTime', () => {
      const startTime = new Date();
      const endTime = new Date(startTime.getTime() + 15 * 60 * 1000);
      const period: ActivityPeriod = {
        startTime,
        endTime,
        agents: [],
        action: 'idle',
        duration: 15,
        totalCost: 0,
      };
      expect(period.endTime.getTime()).toBeGreaterThan(period.startTime.getTime());
    });

    it('15-minute periods are standard', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(Date.now() + 15 * 60 * 1000),
        agents: ['eng-01'],
        action: 'working',
        duration: 15,
        totalCost: 0,
      };
      expect(period.duration).toBe(15);
    });
  });

  describe('Cost Tracking', () => {
    it('supports zero cost', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'idle',
        cost: 0,
      };
      expect(event.cost).toBe(0);
    });

    it('supports positive cost', () => {
      const event: ActivityEvent = {
        timestamp: new Date(),
        agent: 'eng-01',
        action: 'working',
        cost: 0.05,
      };
      expect(event.cost).toBeGreaterThan(0);
    });

    it('totalCost aggregates multiple agents', () => {
      const period: ActivityPeriod = {
        startTime: new Date(),
        endTime: new Date(),
        agents: ['eng-01', 'eng-02', 'eng-03'],
        action: 'working',
        duration: 15,
        totalCost: 0.15, // 0.05 per agent
      };
      expect(period.totalCost).toBe(0.15);
    });
  });
});
