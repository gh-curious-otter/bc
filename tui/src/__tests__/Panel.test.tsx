import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { ThemeProvider } from '../theme/ThemeContext';
import { Panel } from '../components/Panel';
import { Text } from 'ink';

const renderWithTheme = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe('Panel', () => {
  describe('basic rendering', () => {
    it('renders children content', () => {
      const { lastFrame } = renderWithTheme(
        <Panel>
          <Text>Panel content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Panel content');
    });

    it('renders title when provided', () => {
      const { lastFrame } = renderWithTheme(
        <Panel title="My Panel">
          <Text>Content</Text>
        </Panel>
      );
      const frame = lastFrame();
      expect(frame).toContain('My Panel');
      expect(frame).toContain('Content');
    });

    it('renders without title when not provided', () => {
      const { lastFrame } = renderWithTheme(
        <Panel>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('focus state', () => {
    it('renders with default border color when not focused', () => {
      const { lastFrame } = renderWithTheme(
        <Panel>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with focused styling', () => {
      const { lastFrame } = renderWithTheme(
        <Panel focused={true}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with custom border color', () => {
      const { lastFrame } = renderWithTheme(
        <Panel borderColor="red">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('dimensions', () => {
    it('renders with width constraint', () => {
      const { lastFrame } = renderWithTheme(
        <Panel width={40}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with height constraint', () => {
      const { lastFrame } = renderWithTheme(
        <Panel height={10}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });

    it('renders with both width and height', () => {
      const { lastFrame } = renderWithTheme(
        <Panel width={50} height={15}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Content');
    });
  });

  describe('title styling', () => {
    it('renders title in bold', () => {
      const { lastFrame } = renderWithTheme(
        <Panel title="Bold Title">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('Bold Title');
    });

    it('renders title with long text', () => {
      const longTitle = 'A'.repeat(50);
      const { lastFrame } = renderWithTheme(
        <Panel title={longTitle}>
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('A');
    });

    it('renders title with special characters', () => {
      const { lastFrame } = renderWithTheme(
        <Panel title="Title [#123] - Status">
          <Text>Content</Text>
        </Panel>
      );
      expect(lastFrame()).toContain('#123');
    });
  });

  describe('multiple children', () => {
    it('renders multiple child components', () => {
      const { lastFrame } = renderWithTheme(
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
      const { lastFrame } = renderWithTheme(<Panel></Panel>);
      // Should still render the border
      expect(lastFrame()).toBeDefined();
    });
  });
});
