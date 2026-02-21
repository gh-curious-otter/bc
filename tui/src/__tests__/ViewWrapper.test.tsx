/**
 * ViewWrapper component tests
 * Issue #1419: TUI Production Polish
 */

import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { Text } from 'ink';
import { ViewWrapper } from '../components/ViewWrapper';

describe('ViewWrapper', () => {
  test('renders children', () => {
    const { lastFrame } = render(
      <ViewWrapper>
        <Text>Test Content</Text>
      </ViewWrapper>
    );
    expect(lastFrame()).toContain('Test Content');
  });

  test('renders title when provided', () => {
    const { lastFrame } = render(
      <ViewWrapper title="Test Title">
        <Text>Content</Text>
      </ViewWrapper>
    );
    expect(lastFrame()).toContain('Test Title');
  });

  test('shows loading indicator when loading with no children', () => {
    const { lastFrame } = render(
      <ViewWrapper loading loadingMessage="Loading data...">
        {null}
      </ViewWrapper>
    );
    expect(lastFrame()).toContain('Loading data');
  });

  test('shows error display when error is set', () => {
    const { lastFrame } = render(
      <ViewWrapper error="Something went wrong">
        <Text>Content</Text>
      </ViewWrapper>
    );
    expect(lastFrame()).toContain('Something went wrong');
  });

  test('renders footer with hints', () => {
    const { lastFrame } = render(
      <ViewWrapper
        hints={[
          { key: 'j/k', label: 'nav' },
          { key: 'q', label: 'quit' },
        ]}
      >
        <Text>Content</Text>
      </ViewWrapper>
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('j/k');
    expect(output).toContain('nav');
    expect(output).toContain('q');
    expect(output).toContain('quit');
  });

  test('hides footer when hideFooter is true', () => {
    const { lastFrame } = render(
      <ViewWrapper
        hideFooter
        hints={[{ key: 'q', label: 'quit' }]}
      >
        <Text>Content</Text>
      </ViewWrapper>
    );
    expect(lastFrame()).not.toContain('quit');
  });

  test('shows refreshing indicator when loading with content', () => {
    const { lastFrame } = render(
      <ViewWrapper title="Test" loading>
        <Text>Existing Content</Text>
      </ViewWrapper>
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('refreshing');
    expect(output).toContain('Existing Content');
  });

  test('error state takes precedence over loading', () => {
    const { lastFrame } = render(
      <ViewWrapper loading error="Error occurred">
        <Text>Content</Text>
      </ViewWrapper>
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Error occurred');
    expect(output).not.toContain('Loading');
  });

  test('renders custom footer when provided', () => {
    const { lastFrame } = render(
      <ViewWrapper footer={<Text>Custom Footer</Text>}>
        <Text>Content</Text>
      </ViewWrapper>
    );
    expect(lastFrame()).toContain('Custom Footer');
  });

  test('renders with renderWithLayout prop', () => {
    const { lastFrame } = render(
      <ViewWrapper
        renderWithLayout={(layout) => (
          <Text>Width: {layout.width}</Text>
        )}
      />
    );
    // Default ink test width is 80
    expect(lastFrame()).toContain('Width:');
  });
});

describe('ViewWrapper with Panel', () => {
  test('wraps content in Panel when usePanel is true', () => {
    const { lastFrame } = render(
      <ViewWrapper usePanel title="Panel Title">
        <Text>Panel Content</Text>
      </ViewWrapper>
    );
    const output = lastFrame() ?? '';
    // Panel has border characters
    expect(output).toContain('Panel Title');
    expect(output).toContain('Panel Content');
    // Check for border characters (single border style uses these)
    expect(output).toMatch(/[│┌┐└┘─]/);
  });

  test('Panel respects borderColor prop', () => {
    const { lastFrame } = render(
      <ViewWrapper usePanel borderColor="cyan" title="Colored">
        <Text>Content</Text>
      </ViewWrapper>
    );
    // Just verify it renders without error
    expect(lastFrame()).toContain('Colored');
  });
});
