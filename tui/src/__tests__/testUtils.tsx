/**
 * Test utilities for TUI components
 * Issue #1004: Provides test wrappers with necessary providers
 */

import React from 'react';
import { render, type RenderOptions } from 'ink-testing-library';
import { ConfigProvider } from '../config';
import { ThemeProvider } from '../theme';
import { FocusProvider } from '../navigation';

/**
 * Wrapper component that provides all necessary context providers for tests
 */
export function TestProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider config={{ mode: 'dark' }}>
      <ConfigProvider>
        <FocusProvider>
          {children}
        </FocusProvider>
      </ConfigProvider>
    </ThemeProvider>
  );
}

/**
 * Custom render function that wraps components with providers
 */
export function renderWithProviders(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(<TestProviders>{ui}</TestProviders>, options);
}

export { render };
