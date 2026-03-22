/**
 * RolesView component tests
 * Issue #859 - Add Roles tab
 * Issue #1004 - Updated to include ConfigProvider
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, beforeEach, mock } from 'bun:test';
import { ThemeProvider } from '../../theme/ThemeContext';
import { RolesView } from '../../views/RolesView';
import { FocusProvider } from '../../navigation/FocusContext';
import { NavigationProvider } from '../../navigation/NavigationContext';
import { ConfigProvider } from '../../config';
import { DisableInputProvider } from '../../hooks';

// #1594: Helper to wrap component with required providers including DisableInputProvider
// #1604: Add NavigationProvider for breadcrumb context
const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <ThemeProvider>
      <ConfigProvider>
        <NavigationProvider>
          <FocusProvider>
            <DisableInputProvider disabled>{ui}</DisableInputProvider>
          </FocusProvider>
        </NavigationProvider>
      </ConfigProvider>
    </ThemeProvider>
  );
};

// Legacy alias for backward compatibility
const renderWithFocus = renderWithProviders;

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
      const { lastFrame } = renderWithFocus(<RolesView />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders loading state initially', () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      // Initial state shows loading
      expect(lastFrame()).toBeDefined();
    });

    it('renders with disableInput prop', () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('role list display', () => {
    it('shows role names after loading', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('engineer');
    });

    it('shows manager role', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('manager');
    });

    it('shows tech-lead role', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('tech-lead');
    });
  });

  describe('table headers', () => {
    it('shows NAME column', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('NAME');
    });

    it('shows CAPABILITIES column', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('CAPABILITIES');
    });

    it('shows AGENTS column', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('AGENTS');
    });

    it('shows DESCRIPTION column', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('DESCRIPTION');
    });
  });

  describe('search bar', () => {
    it('shows search hint', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('search');
    });

    it('shows navigation hint', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('j/k');
    });
  });

  describe('footer', () => {
    it('shows navigate hint', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('navigate');
    });

    it('shows Enter hint for details', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('details');
    });

    it('shows refresh hint', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('refresh');
    });

    it('shows back hint', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('back');
    });
  });

  describe('role count', () => {
    it('shows total role count', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      // Should show (3) for 3 roles
      expect(output).toContain('3');
    });
  });

  describe('capabilities display', () => {
    it('shows capabilities in row', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      expect(output).toContain('implement');
    });
  });

  describe('selection indicator', () => {
    it('shows selection marker', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame();
      // First item should be selected with marker
      expect(output).toContain('\u25b8');
    });
  });

  // #971 fix: Dynamic name column width tests
  describe('dynamic name column width', () => {
    it('shows full role name without truncation for short names', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame() ?? '';
      // 'tech-lead' is 9 chars, should not be truncated
      expect(output).toContain('tech-lead');
      expect(output).not.toContain('tech-lea…');
    });

    it('shows role name without ellipsis when within bounds', async () => {
      const { lastFrame } = renderWithFocus(<RolesView />);
      await new Promise((r) => setTimeout(r, 150));
      const output = lastFrame() ?? '';
      // 'engineer' and 'manager' are short, should not have ellipsis
      expect(output).toContain('engineer');
      expect(output).toContain('manager');
    });
  });
});
