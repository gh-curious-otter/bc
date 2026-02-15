/**
 * Test Utilities - Common testing helpers for TUI components
 *
 * Provides:
 * - renderWithProviders: Render with theme and context providers
 * - mockBcService: Mock bc CLI service responses
 * - Factory functions: Create mock data (agents, channels, etc.)
 * - Keyboard simulation: Simulate user input
 * - Async helpers: Wait for elements and conditions
 */

import React, { type ReactElement } from 'react';
import { render as inkRender } from 'ink-testing-library';
import { ThemeProvider } from '../../theme/ThemeContext';
import { FocusProvider } from '../../navigation/FocusContext';
import { NavigationProvider } from '../../navigation/NavigationContext';
import type { ThemeConfig } from '../../theme/types';
import type { FocusArea } from '../../navigation/FocusContext';

/**
 * renderWithProviders - Render component with all required providers
 *
 * Wraps component with:
 * - ThemeProvider (for color system)
 * - FocusProvider (for keyboard focus state)
 * - NavigationProvider (for view/tab navigation)
 *
 * @param component React component to render
 * @param options Configuration options
 * @returns Render result from ink-testing-library
 */
export function renderWithProviders(
  component: ReactElement,
  options?: {
    themeConfig?: ThemeConfig;
    initialView?: string;
    initialFocus?: FocusArea;
    disableInput?: boolean;
  }
) {
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <ThemeProvider config={options?.themeConfig}>
      <FocusProvider initialFocus={options?.initialFocus || 'main'}>
        <NavigationProvider>
          {children}
        </NavigationProvider>
      </FocusProvider>
    </ThemeProvider>
  );

  return inkRender(<Wrapper>{component}</Wrapper>);
}

/**
 * mockBcService - Create a mock bc CLI service
 *
 * Returns a mock implementation that:
 * - Returns configured responses
 * - Tracks call history
 * - Can simulate delays
 * - Can trigger errors
 *
 * @param responses Configured responses for bc commands
 * @returns Mock service instance
 */
export function mockBcService(responses?: Record<string, any>) {
  const callHistory: Array<{ command: string; args: string[] }> = [];

  const mockService = {
    /**
     * Execute a bc command
     */
    execute: async (command: string, args: string[] = []) => {
      callHistory.push({ command, args });

      // Return configured response if available
      if (responses && responses[command]) {
        return responses[command];
      }

      // Return default empty response
      return { success: true, data: null };
    },

    /**
     * Get call history
     */
    getCallHistory: () => [...callHistory],

    /**
     * Clear call history
     */
    clearHistory: () => callHistory.splice(0, callHistory.length),

    /**
     * Assert command was called
     */
    assertCalled: (command: string) => {
      const called = callHistory.some(call => call.command === command);
      if (!called) {
        throw new Error(`Expected command "${command}" to be called, but it wasn't`);
      }
    },

    /**
     * Assert command was NOT called
     */
    assertNotCalled: (command: string) => {
      const called = callHistory.some(call => call.command === command);
      if (called) {
        throw new Error(`Expected command "${command}" not to be called, but it was`);
      }
    },
  };

  return mockService;
}

/**
 * Simulate keyboard input
 *
 * @param key Key to simulate (e.g., 'enter', 'escape', 'j', 'k')
 * @returns Simulated keyboard input object
 */
export function simulateKeypress(key: string) {
  const keyMap: Record<string, any> = {
    'enter': { return: true },
    'escape': { escape: true },
    'backspace': { backspace: true },
    'tab': { tab: true },
    'arrowup': { upArrow: true },
    'arrowdown': { downArrow: true },
    'arrowleft': { leftArrow: true },
    'arrowright': { rightArrow: true },
  };

  // Single character key
  if (key.length === 1) {
    return { input: key, key: {} };
  }

  // Special key
  return { input: '', key: keyMap[key.toLowerCase()] || {} };
}

/**
 * Wait for an element matching predicate
 *
 * Polls until element appears or timeout
 *
 * @param predicate Function that returns true when element is found
 * @param timeout Max wait time in ms (default: 1000)
 * @param interval Poll interval in ms (default: 50)
 */
export async function waitForElement(
  predicate: () => boolean,
  timeout = 1000,
  interval = 50
): Promise<void> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeout) {
    if (predicate()) {
      return;
    }
    await new Promise(resolve => setTimeout(resolve, interval));
  }

  throw new Error(`Element not found within ${timeout}ms`);
}

/**
 * Wait for text to appear in output
 *
 * @param text Text to search for
 * @param frameGetter Function that returns frame output
 * @param timeout Max wait time in ms
 */
export async function waitForText(
  text: string,
  frameGetter: () => string,
  timeout = 1000
): Promise<void> {
  return waitForElement(() => frameGetter().includes(text), timeout);
}

/**
 * Create a test component that renders a hook
 *
 * Useful for testing custom hooks in isolation
 *
 * @param hook Custom hook to test
 * @param initialProps Initial props for hook
 * @returns Test component
 */
export function createHookTestComponent<T, P>(
  hook: (props: P) => T,
  initialProps: P
) {
  let result: T;

  const TestComponent = () => {
    result = hook(initialProps);
    return <></>;
  };

  return { TestComponent, getResult: () => result };
}

export default {
  renderWithProviders,
  mockBcService,
  simulateKeypress,
  waitForElement,
  waitForText,
  createHookTestComponent,
};
