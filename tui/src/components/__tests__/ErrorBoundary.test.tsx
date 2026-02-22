import { describe, it, expect, vi, beforeEach, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { ErrorBoundary, ViewErrorBoundary } from '../ErrorBoundary';
import { Text } from 'ink';

// Component that throws an error for testing
function ErrorThrowingComponent(): React.ReactElement {
  throw new Error('Test error from component');
}

// Component that works normally
function WorkingComponent(): React.ReactElement {
  return <Text>Working content</Text>;
}

describe('ErrorBoundary', () => {
  // Suppress console.error during tests
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it('renders children when no error occurs', () => {
    const { lastFrame } = render(
      <ErrorBoundary>
        <WorkingComponent />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain('Working content');
  });

  it('renders error message when child throws', () => {
    const { lastFrame } = render(
      <ErrorBoundary>
        <ErrorThrowingComponent />
      </ErrorBoundary>
    );

    const frame = lastFrame();
    expect(frame).toContain('Error');
    expect(frame).toContain('Test error from component');
  });

  it('includes view name in error message when provided', () => {
    const { lastFrame } = render(
      <ErrorBoundary viewName="Dashboard">
        <ErrorThrowingComponent />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain('Error in Dashboard');
  });

  it('shows navigation hint in error state', () => {
    const { lastFrame } = render(
      <ErrorBoundary>
        <ErrorThrowingComponent />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain('navigation keys');
  });

  it('renders custom fallback when provided', () => {
    const { lastFrame } = render(
      <ErrorBoundary fallback={<Text>Custom fallback</Text>}>
        <ErrorThrowingComponent />
      </ErrorBoundary>
    );

    expect(lastFrame()).toContain('Custom fallback');
    expect(lastFrame()).not.toContain('Error in');
  });

  it('logs error to console', () => {
    render(
      <ErrorBoundary viewName="TestView">
        <ErrorThrowingComponent />
      </ErrorBoundary>
    );

    expect(consoleErrorSpy).toHaveBeenCalled();
    const calls = consoleErrorSpy.mock.calls;
    const hasErrorLog = calls.some(
      (call) => typeof call[0] === 'string' && call[0].includes('ErrorBoundary')
    );
    expect(hasErrorLog).toBe(true);
  });
});

describe('ViewErrorBoundary', () => {
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it('renders children when no error', () => {
    const { lastFrame } = render(
      <ViewErrorBoundary viewName="agents">
        <WorkingComponent />
      </ViewErrorBoundary>
    );

    expect(lastFrame()).toContain('Working content');
  });

  it('catches errors with view name', () => {
    const { lastFrame } = render(
      <ViewErrorBoundary viewName="agents">
        <ErrorThrowingComponent />
      </ViewErrorBoundary>
    );

    expect(lastFrame()).toContain('Error in agents');
    expect(lastFrame()).toContain('Test error from component');
  });
});
