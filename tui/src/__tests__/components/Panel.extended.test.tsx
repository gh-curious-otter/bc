/**
 * Panel component extended tests
 * Issue #682 - Component Testing
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Text } from 'ink';
import { describe, it, expect } from 'bun:test';
import { Panel } from '../../components/Panel';

describe('Panel - Extended Tests', () => {
  describe('title variations', () => {
    it('renders without title', () => {
      const { lastFrame } = render(
        <Panel>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with short title', () => {
      const { lastFrame } = render(
        <Panel title="Hi">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Hi');
    });

    it('renders with long title', () => {
      const longTitle = 'A'.repeat(50);
      const { lastFrame } = render(
        <Panel title={longTitle}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('A');
    });

    it('renders with special characters in title', () => {
      const { lastFrame } = render(
        <Panel title="Test: Special!@#$%">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with unicode title', () => {
      const { lastFrame } = render(
        <Panel title="测试 🎉">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('测试');
    });

    it('renders with empty string title', () => {
      const { lastFrame } = render(
        <Panel title="">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('border styles', () => {
    it('renders with default border color', () => {
      const { lastFrame } = render(
        <Panel title="Test">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with custom border color', () => {
      const { lastFrame } = render(
        <Panel title="Test" borderColor="red">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders when focused', () => {
      const { lastFrame } = render(
        <Panel title="Test" focused>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders when not focused', () => {
      const { lastFrame } = render(
        <Panel title="Test" focused={false}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('dimensions', () => {
    it('renders with fixed width', () => {
      const { lastFrame } = render(
        <Panel title="Test" width={50}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with fixed height', () => {
      const { lastFrame } = render(
        <Panel title="Test" height={10}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with both fixed dimensions', () => {
      const { lastFrame } = render(
        <Panel title="Test" width={50} height={10}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with percentage width', () => {
      const { lastFrame } = render(
        <Panel title="Test" width="50%">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with small width', () => {
      const { lastFrame } = render(
        <Panel title="Test" width={10}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('children content', () => {
    it('renders single child', () => {
      const { lastFrame } = render(
        <Panel title="Test">
          <Text>Single child</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Single child');
    });

    it('renders multiple children', () => {
      const { lastFrame } = render(
        <Panel title="Test">
          <Text>Child 1</Text>
          <Text>Child 2</Text>
          <Text>Child 3</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Child 1');
      expect(lastFrame()).toContain('Child 2');
      expect(lastFrame()).toContain('Child 3');
    });

    it('renders nested panels', () => {
      const { lastFrame } = render(
        <Panel title="Outer">
          <Panel title="Inner">
            <Text>Nested content</Text>
          </Panel>
        </Panel>
      );
      expect(lastFrame()).toContain('Outer');
      expect(lastFrame()).toContain('Inner');
    });

    it('renders with no children', () => {
      const { lastFrame } = render(<Panel title="Empty">{null}</Panel>);
      expect(lastFrame()).toContain('Empty');
    });

    it('renders with complex children', () => {
      const { lastFrame } = render(
        <Panel title="Complex">
          <Text bold>Bold text</Text>
          <Text color="red">Red text</Text>
          <Text dimColor>Dim text</Text>
        </Panel>
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('consistency', () => {
    it('produces consistent output', () => {
      const { lastFrame: frame1 } = render(
        <Panel title="Test">
          <Text>Content</Text>
        </Panel>
      );
      const { lastFrame: frame2 } = render(
        <Panel title="Test">
          <Text>Content</Text>
        </Panel>
      );
      expect(frame1()).toBe(frame2());
    });
  });
});
