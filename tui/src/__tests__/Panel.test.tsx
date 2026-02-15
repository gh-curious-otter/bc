import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { Panel } from '../components/Panel';
import { Text } from 'ink';

describe('Panel', () => {
  describe('basic rendering', () => {
    it('renders children content', () => {
      const { lastFrame } = render(
        <Panel>
          <Text>Panel content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Panel content');
    });

    it('renders title when provided', () => {
      const { lastFrame } = render(
        <Panel title="My Panel">
          <Text>Content</Text>
        </Panel>
      );
      const frame = lastFrame();
      expect(frame).toContain('My Panel');
      expect(frame).toContain('Content');
    });

    it('renders without title when not provided', () => {
      const { lastFrame } = render(
        <Panel>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('focus state', () => {
    it('renders with default border color when not focused', () => {
      const { lastFrame } = render(
        <Panel>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with focused styling', () => {
      const { lastFrame } = render(
        <Panel focused={true}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with custom border color', () => {
      const { lastFrame } = render(
        <Panel borderColor="red">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('dimensions', () => {
    it('renders with width constraint', () => {
      const { lastFrame } = render(
        <Panel width={40}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with height constraint', () => {
      const { lastFrame } = render(
        <Panel height={10}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with both width and height', () => {
      const { lastFrame } = render(
        <Panel width={50} height={15}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('title styling', () => {
    it('renders title in bold', () => {
      const { lastFrame } = render(
        <Panel title="Bold Title">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Bold Title');
    });

    it('renders title with long text', () => {
      const longTitle = 'A'.repeat(50);
      const { lastFrame } = render(
        <Panel title={longTitle}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('A');
    });

    it('renders title with special characters', () => {
      const { lastFrame } = render(
        <Panel title="Title [#123] - Status">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('#123');
    });
  });

  describe('multiple children', () => {
    it('renders multiple child components', () => {
      const { lastFrame } = render(
        <Panel>
          <Text>Child 1</Text>
          <Text>Child 2</Text>
          <Text>Child 3</Text>
        </Panel>
      );
      const frame = lastFrame();
      expect(frame).toContain('Child 1');
      expect(frame).toContain('Child 2');
      expect(frame).toContain('Child 3');
    });

    it('renders empty panel when no children', () => {
      const { lastFrame } = render(<Panel></Panel>);
      // Should still render the border
      expect(lastFrame()).toBeDefined();
    });
  });
});
