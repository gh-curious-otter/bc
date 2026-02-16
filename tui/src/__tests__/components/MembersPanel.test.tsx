/**
 * MembersPanel component tests
 * Issue #847 - Channel member list + description
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { MembersPanel, MemberCountBadge } from '../../components/MembersPanel';
import type { MemberInfo } from '../../components/MembersPanel';

const mockMembers: string[] = ['eng-01', 'eng-02', 'tl-01', 'mgr-01'];

const mockMembersWithInfo: MemberInfo[] = [
  { name: 'eng-01', role: 'engineer', state: 'working' },
  { name: 'eng-02', role: 'engineer', state: 'idle' },
  { name: 'tl-01', role: 'tech-lead', state: 'working' },
];

describe('MembersPanel', () => {
  describe('basic rendering', () => {
    it('renders member list', () => {
      const { lastFrame } = render(<MembersPanel members={mockMembers} />);
      const output = lastFrame();
      expect(output).toContain('eng-01');
      expect(output).toContain('eng-02');
    });

    it('shows member count in title', () => {
      const { lastFrame } = render(<MembersPanel members={mockMembers} />);
      const output = lastFrame();
      expect(output).toContain('(4)');
    });

    it('renders with custom title', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} title="Channel Members" />
      );
      const output = lastFrame();
      expect(output).toContain('Channel Members');
    });

    it('renders empty list', () => {
      const { lastFrame } = render(<MembersPanel members={[]} />);
      const output = lastFrame();
      expect(output).toContain('(0)');
    });
  });

  describe('member info display', () => {
    it('renders string members', () => {
      const { lastFrame } = render(<MembersPanel members={mockMembers} />);
      const output = lastFrame();
      expect(output).toContain('eng-01');
      expect(output).toContain('tl-01');
    });

    it('renders members with role info', () => {
      const { lastFrame } = render(<MembersPanel members={mockMembersWithInfo} />);
      const output = lastFrame();
      expect(output).toContain('eng-01');
      expect(output).toContain('engineer');
    });

    it('renders members with state info', () => {
      const { lastFrame } = render(<MembersPanel members={mockMembersWithInfo} />);
      const output = lastFrame();
      expect(output).toContain('working');
    });
  });

  describe('collapsible behavior', () => {
    it('renders expanded by default', () => {
      const { lastFrame } = render(<MembersPanel members={mockMembers} />);
      const output = lastFrame();
      expect(output).toContain('eng-01');
    });

    it('renders collapsed when defaultCollapsed is true', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} defaultCollapsed />
      );
      const output = lastFrame();
      expect(output).toContain('expand');
    });

    it('shows collapse hint when collapsible', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} collapsible />
      );
      const output = lastFrame();
      expect(output).toContain('collapse');
    });

    it('hides collapse hint when not collapsible', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} collapsible={false} />
      );
      const output = lastFrame();
      expect(output).not.toContain('Press space');
    });
  });

  describe('maxVisible limit', () => {
    const manyMembers = Array.from({ length: 20 }, (_, i) => `member-${String(i + 1)}`);

    it('limits visible members', () => {
      const { lastFrame } = render(
        <MembersPanel members={manyMembers} maxVisible={5} />
      );
      const output = lastFrame();
      expect(output).toContain('member-1');
      expect(output).toContain('member-5');
      expect(output).toContain('and 15 more');
    });

    it('shows all members when under limit', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} maxVisible={10} />
      );
      const output = lastFrame();
      expect(output).not.toContain('more');
    });

    it('handles exact limit', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} maxVisible={4} />
      );
      const output = lastFrame();
      expect(output).not.toContain('more');
    });
  });

  describe('focus state', () => {
    it('renders without focus', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} focused={false} />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with focus', () => {
      const { lastFrame } = render(
        <MembersPanel members={mockMembers} focused />
      );
      expect(lastFrame()).toBeDefined();
    });
  });
});

describe('MemberCountBadge', () => {
  it('renders count', () => {
    const { lastFrame } = render(<MemberCountBadge count={5} />);
    const output = lastFrame();
    expect(output).toContain('[5]');
  });

  it('renders zero count', () => {
    const { lastFrame } = render(<MemberCountBadge count={0} />);
    const output = lastFrame();
    expect(output).toContain('[0]');
  });

  it('renders large count', () => {
    const { lastFrame } = render(<MemberCountBadge count={100} />);
    const output = lastFrame();
    expect(output).toContain('[100]');
  });

  it('renders with custom color', () => {
    const { lastFrame } = render(<MemberCountBadge count={3} color="green" />);
    expect(lastFrame()).toBeDefined();
  });

  it('renders with default color', () => {
    const { lastFrame } = render(<MemberCountBadge count={3} />);
    expect(lastFrame()).toBeDefined();
  });
});
