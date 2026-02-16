/**
 * RolesView component tests
 * Issue #859 - Add Roles tab
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, beforeEach, mock } from 'bun:test';
import { RolesView } from '../../views/RolesView';

// Mock the bc service
mock.module('../../services/bc', () => ({
  getRoles: vi.fn().mockResolvedValue({
    roles: [
      {
        name: 'engineer',
        description: 'Engineering role',
        capabilities: ['implement_tasks', 'test_code'],
        agent_count: 3,
      },
      {
        name: 'manager',
        description: 'Management role',
        capabilities: ['assign_work', 'review_code'],
        agent_count: 1,
      },
      {
        name: 'tech-lead',
        description: 'Tech lead role',
        capabilities: ['architect', 'review_code'],
        agent_count: 2,
      },
    ],
  }),
  getRole: vi.fn().mockResolvedValue({
    name: 'engineer',
    description: 'Engineering role',
    capabilities: ['implement_tasks', 'test_code'],
    prompt: 'You are an engineer...',
    agent_count: 3,
  }),
  deleteRole: vi.fn().mockResolvedValue(undefined),
}));

describe('RolesView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('basic rendering', () => {
    it('renders without crashing', () => {
      const { lastFrame } = render(<RolesView disableInput />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders loading state initially', () => {
      const { lastFrame } = render(<RolesView disableInput />);
      // Initial state shows loading
      expect(lastFrame()).toBeDefined();
    });

    it('renders with disableInput prop', () => {
      const { lastFrame } = render(<RolesView disableInput />);
      expect(lastFrame()).toBeDefined();
    });

    it('accepts onBack callback', () => {
      const onBack = vi.fn();
      const { lastFrame } = render(<RolesView onBack={onBack} disableInput />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('role list display', () => {
    it('shows role names after loading', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('engineer');
    });

    it('shows manager role', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('manager');
    });

    it('shows tech-lead role', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('tech-lead');
    });
  });

  describe('table headers', () => {
    it('shows NAME column', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('NAME');
    });

    it('shows CAPABILITIES column', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('CAPABILITIES');
    });

    it('shows AGENTS column', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('AGENTS');
    });

    it('shows DESCRIPTION column', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('DESCRIPTION');
    });
  });

  describe('search bar', () => {
    it('shows search hint', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('search');
    });

    it('shows navigation hint', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('j/k');
    });
  });

  describe('footer', () => {
    it('shows navigate hint', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('navigate');
    });

    it('shows Enter hint for details', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('details');
    });

    it('shows refresh hint', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('refresh');
    });

    it('shows back hint', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('back');
    });
  });

  describe('role count', () => {
    it('shows total role count', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      // Should show (3) for 3 roles
      expect(output).toContain('3');
    });
  });

  describe('capabilities display', () => {
    it('shows capabilities in row', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('implement');
    });
  });

  describe('selection indicator', () => {
    it('shows selection marker', async () => {
      const { lastFrame } = render(<RolesView disableInput />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      // First item should be selected with marker
      expect(output).toContain('\u25b8');
    });
  });
});
