/**
 * InlineEditor component tests
 * Issue #858 - Inline editing
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi } from 'bun:test';
import { ThemeProvider } from '../../theme/ThemeContext';
import { InlineEditor, EditorModal } from '../../components/InlineEditor';

const renderWithTheme = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe('InlineEditor', () => {
  describe('basic rendering', () => {
    it('renders without crashing', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor disableInput />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with initial value', () => {
      const { lastFrame } = renderWithTheme(
        <InlineEditor initialValue="Hello" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Hello');
    });

    it('renders placeholder when empty', () => {
      const { lastFrame } = renderWithTheme(
        <InlineEditor placeholder="Type here" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Type here');
    });

    it('renders with custom placeholder', () => {
      const { lastFrame } = renderWithTheme(
        <InlineEditor placeholder="Enter name" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Enter name');
    });

    it('renders with label', () => {
      const { lastFrame } = renderWithTheme(
        <InlineEditor label="Name" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Name');
    });
  });

  describe('single-line mode', () => {
    it('shows single-line hints', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor disableInput />);
      const output = lastFrame();
      expect(output).toContain('Enter');
      expect(output).toContain('save');
    });

    it('shows cancel hint', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor disableInput />);
      const output = lastFrame();
      expect(output).toContain('Esc');
      expect(output).toContain('cancel');
    });

    it('renders border', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor disableInput />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('multi-line mode', () => {
    it('renders in multi-line mode', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor multiline disableInput />);
      expect(lastFrame()).toBeDefined();
    });

    it('shows multi-line hints', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor multiline disableInput />);
      const output = lastFrame();
      expect(output).toContain('Ctrl+S');
      expect(output).toContain('newline');
    });

    it('renders multi-line content', () => {
      const { lastFrame } = renderWithTheme(
        <InlineEditor
          initialValue={'Line 1\nLine 2\nLine 3'}
          multiline
          disableInput
        />
      );
      const output = lastFrame();
      expect(output).toContain('Line 1');
      expect(output).toContain('Line 2');
    });

    it('respects maxHeight', () => {
      const longContent = Array(20).fill('Line').join('\n');
      const { lastFrame } = renderWithTheme(
        <InlineEditor
          initialValue={longContent}
          multiline
          maxHeight={5}
          disableInput
        />
      );
      const output = lastFrame();
      expect(output).toContain('more lines');
    });
  });

  describe('focus state', () => {
    it('renders focused state', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor focused disableInput />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders unfocused state', () => {
      const { lastFrame } = renderWithTheme(<InlineEditor focused={false} disableInput />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('callbacks', () => {
    it('accepts onChange callback', () => {
      const onChange = vi.fn();
      const { lastFrame } = renderWithTheme(
        <InlineEditor onChange={onChange} disableInput />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('accepts onSave callback', () => {
      const onSave = vi.fn();
      const { lastFrame } = renderWithTheme(
        <InlineEditor onSave={onSave} disableInput />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('accepts onCancel callback', () => {
      const onCancel = vi.fn();
      const { lastFrame } = renderWithTheme(
        <InlineEditor onCancel={onCancel} disableInput />
      );
      expect(lastFrame()).toBeDefined();
    });
  });
});

describe('EditorModal', () => {
  describe('visibility', () => {
    it('renders nothing when not visible', () => {
      const { lastFrame } = renderWithTheme(
        <EditorModal visible={false} disableInput />
      );
      expect(lastFrame()).toBe('');
    });

    it('renders when visible', () => {
      const { lastFrame } = renderWithTheme(<EditorModal visible disableInput />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('title', () => {
    it('renders with default title', () => {
      const { lastFrame } = renderWithTheme(<EditorModal visible disableInput />);
      const output = lastFrame();
      expect(output).toContain('Edit');
    });

    it('renders with custom title', () => {
      const { lastFrame } = renderWithTheme(
        <EditorModal visible title="Edit Role" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Edit Role');
    });
  });

  describe('editor content', () => {
    it('passes initial value to editor', () => {
      const { lastFrame } = renderWithTheme(
        <EditorModal visible initialValue="Test value" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Test value');
    });

    it('passes label to editor', () => {
      const { lastFrame } = renderWithTheme(
        <EditorModal visible label="Field" disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Field');
    });

    it('supports multiline in modal', () => {
      const { lastFrame } = renderWithTheme(
        <EditorModal visible multiline disableInput />
      );
      const output = lastFrame();
      expect(output).toContain('Ctrl+S');
    });
  });
});
