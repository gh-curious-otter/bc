import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { ThemeProvider } from '../theme/ThemeContext';
import { ErrorDisplay } from '../components/ErrorDisplay';

const renderWithTheme = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe('ErrorDisplay', () => {
  describe('error message rendering', () => {
    it('renders string error message', () => {
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error="Something went wrong" />
      );
      expect(lastFrame()).toContain('Error');
      expect(lastFrame()).toContain('Something went wrong');
    });

    it('renders Error object message', () => {
      const error = new Error('Failed to load data');
      const { lastFrame } = renderWithTheme(<ErrorDisplay error={error} />);
      expect(lastFrame()).toContain('Error');
      expect(lastFrame()).toContain('Failed to load data');
    });

    it('renders error with special characters', () => {
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error="Error: Failed @ operation #123 with status 500" />
      );
      expect(lastFrame()).toContain('Error');
      expect(lastFrame()).toContain('@');
    });
  });

  describe('retry option', () => {
    it('does not show retry message when onRetry is not provided', () => {
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error="Network error" />
      );
      const frame = lastFrame();
      expect(frame).toContain('Network error');
      expect(frame).not.toContain('retry');
    });

    it('shows retry message when onRetry is provided', () => {
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error="Network error" onRetry={() => {}} />
      );
      expect(lastFrame()).toContain('retry');
    });
  });

  describe('edge cases', () => {
    it('handles empty error message', () => {
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error="" />
      );
      expect(lastFrame()).toContain('Error');
    });

    it('handles long error messages', () => {
      const longError = 'A'.repeat(200);
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error={longError} />
      );
      expect(lastFrame()).toContain('A');
    });

    it('handles multiline error messages', () => {
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error="Line 1\nLine 2\nLine 3" />
      );
      const frame = lastFrame();
      expect(frame).toContain('Line 1');
    });

    it('handles errors with custom properties', () => {
      const error = new Error('Custom error');
      (error as any).code = 'ERR_CUSTOM';
      const { lastFrame } = renderWithTheme(
        <ErrorDisplay error={error} />
      );
      expect(lastFrame()).toContain('Custom error');
    });
  });
});
