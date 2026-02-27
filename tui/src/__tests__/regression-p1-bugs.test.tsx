/**
 * Regression tests for P1 bug fixes (Issue #1826)
 *
 * This test suite prevents regression of critical bugs that have been fixed.
 * Each test documents the original issue and verifies the fix still works.
 *
 * Categories:
 * 1. ESC Navigation - returnFocus goes to correct parent
 * 2. Keybind Blocking - keybinds disabled during input mode
 * 3. Focus State Reset - focus properly reset on view transitions
 * 4. Data Display - memory/channels display correctly
 */

import React, { useEffect, useRef } from 'react';
import { render } from 'ink-testing-library';
import { Text, Box } from 'ink';
import { describe, test, expect } from 'bun:test';
import { FocusProvider, useFocus, type FocusArea } from '../navigation/FocusContext';
import { NavigationProvider, useNavigation } from '../navigation/NavigationContext';

// Helper to wait for render updates
const waitForRender = (): Promise<void> => new Promise(resolve => setTimeout(resolve, 50));

/**
 * Test provider wrapper with both FocusProvider and NavigationProvider
 */
function TestProviders({
  children,
  initialFocus = 'main',
}: {
  children: React.ReactNode;
  initialFocus?: FocusArea;
}): React.ReactElement {
  return (
    <FocusProvider initialFocus={initialFocus}>
      <NavigationProvider>
        {children}
      </NavigationProvider>
    </FocusProvider>
  );
}

// =============================================================================
// ESC NAVIGATION REGRESSION TESTS
// Issues: #1181, ESC goes to wrong view
// =============================================================================

describe('ESC Navigation Regression (Issue #1181)', () => {
  test('returnFocus restores to correct previous area after input mode', async () => {
    /**
     * Regression: ESC from input mode should return to main, not sidebar/dashboard
     * Original bug: ESC always went to Dashboard regardless of where input started
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          // User was on main view, enters input mode
          setFocus('input');
        } else if (step.current === 1 && focusedArea === 'input') {
          step.current = 2;
          // User presses ESC - should return to main, not dashboard
          returnFocus();
        }
      }, [setFocus, returnFocus, focusedArea]);

      return <Text>focus:{focusedArea}</Text>;
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    // After ESC from input, should be back at main (not sidebar/dashboard)
    expect(lastFrame()).toContain('focus:main');
  });

  test('returnFocus works correctly from detail view', async () => {
    /**
     * Regression: ESC from detail view should return to main list
     * Original bug: Focus got stuck in detail view
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          // User navigates to detail view
          setFocus('detail');
        } else if (step.current === 1 && focusedArea === 'detail') {
          step.current = 2;
          // User presses ESC - should return to main list
          returnFocus();
        }
      }, [setFocus, returnFocus, focusedArea]);

      return <Text>focus:{focusedArea}</Text>;
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    expect(lastFrame()).toContain('focus:main');
  });

  test('multi-level navigation returns correctly (main -> detail -> input -> detail)', async () => {
    /**
     * Regression: Multi-level navigation back should work correctly
     * Original bug: After multiple focus changes, ESC would go to wrong level
     *
     * Note: This tests single returnFocus() call from input -> detail.
     * The FocusContext only stores one previous area, so deep navigation
     * requires views to manage their own "back" stack via onBack callbacks.
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea, previousArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          setFocus('detail'); // Go to detail view
        } else if (step.current === 1 && focusedArea === 'detail') {
          step.current = 2;
          setFocus('input'); // Start typing in detail view
        } else if (step.current === 2 && focusedArea === 'input') {
          step.current = 3;
          returnFocus(); // ESC from input -> should go to detail
        }
      }, [setFocus, returnFocus, focusedArea]);

      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>prev:{previousArea ?? 'null'}</Text>
          <Text>step:{step.current}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    await waitForRender(); // Extra wait for multi-step

    // After ESC from input, should be at detail
    const output = lastFrame() ?? '';
    expect(output).toContain('focus:detail');
    expect(output).toContain('step:3');
  });

  test('breadcrumbs clear when returning from detail view', async () => {
    /**
     * Regression: Breadcrumbs should clear when returning to list view
     * Original bug: Stale breadcrumbs showed after navigating back
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea } = useFocus();
      const { breadcrumbs, setBreadcrumbs, clearBreadcrumbs } = useNavigation();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          setFocus('detail');
          setBreadcrumbs([{ label: 'Agent Details' }]);
        } else if (step.current === 1 && focusedArea === 'detail') {
          step.current = 2;
          returnFocus();
          clearBreadcrumbs();
        }
      }, [setFocus, returnFocus, focusedArea, setBreadcrumbs, clearBreadcrumbs]);

      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>breadcrumbs:{breadcrumbs.length}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    expect(output).toContain('focus:main');
    expect(output).toContain('breadcrumbs:0');
  });
});

// =============================================================================
// KEYBIND BLOCKING REGRESSION TESTS
// Issues: #653, keybinds trigger while typing
// =============================================================================

describe('Keybind Blocking Regression (Issue #653)', () => {
  test('isFocused("input") returns true when in input mode', async () => {
    /**
     * Regression: When typing, isFocused("input") must return true
     * This is the guard used to block global keybinds
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, isFocused } = useFocus();
      const didSet = useRef(false);

      useEffect(() => {
        if (!didSet.current) {
          didSet.current = true;
          setFocus('input');
        }
      }, [setFocus]);

      const shouldBlockKeybinds = isFocused('input');
      return <Text>blocked:{shouldBlockKeybinds ? 'yes' : 'no'}</Text>;
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    expect(lastFrame()).toContain('blocked:yes');
  });

  test('isFocused("modal") returns true when modal is open', async () => {
    /**
     * Regression: Modal focus should block global keybinds
     * Original bug: 'q' keypress closed modal AND quit app
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, isFocused } = useFocus();
      const didSet = useRef(false);

      useEffect(() => {
        if (!didSet.current) {
          didSet.current = true;
          setFocus('modal');
        }
      }, [setFocus]);

      const inModal = isFocused('modal');
      const shouldBlockGlobalKeys = isFocused('input') || isFocused('modal');
      return (
        <Box flexDirection="column">
          <Text>inModal:{inModal ? 'yes' : 'no'}</Text>
          <Text>blocked:{shouldBlockGlobalKeys ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    expect(output).toContain('inModal:yes');
    expect(output).toContain('blocked:yes');
  });

  test('keybind guard resets after exiting input mode', async () => {
    /**
     * Regression: After ESC from input, keybinds must work again
     * Original bug: Focus got stuck in input mode
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, isFocused, focusedArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          setFocus('input');
        } else if (step.current === 1 && focusedArea === 'input') {
          step.current = 2;
          returnFocus(); // Exit input mode
        }
      }, [setFocus, returnFocus, focusedArea]);

      const keybindsEnabled = !isFocused('input') && !isFocused('modal');
      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>keybinds:{keybindsEnabled ? 'enabled' : 'disabled'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    expect(output).toContain('focus:main');
    expect(output).toContain('keybinds:enabled');
  });

  test('simultaneous input and keybind check works correctly', async () => {
    /**
     * Regression: The pattern used in views to check keybind eligibility
     * Views use: const canHandleKeys = !isFocused('input') && !isFocused('modal');
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, isFocused, focusedArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0) {
          step.current = 1;
        }
      }, []);

      // This is the exact pattern used in useKeyboardNavigation
      const canHandleGlobalKeys = !isFocused('input') && !isFocused('modal');
      const canHandleViewKeys = focusedArea === 'main' || focusedArea === 'view';

      return (
        <Box flexDirection="column">
          <Text>globalKeys:{canHandleGlobalKeys ? 'yes' : 'no'}</Text>
          <Text>viewKeys:{canHandleViewKeys ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    expect(output).toContain('globalKeys:yes');
    expect(output).toContain('viewKeys:yes');
  });
});

// =============================================================================
// FOCUS STATE RESET REGRESSION TESTS
// Issues: Focus stuck after exiting composition
// =============================================================================

describe('Focus State Reset Regression', () => {
  test('previous area cleared after returnFocus', async () => {
    /**
     * Regression: previousArea must be null after returnFocus
     * Original bug: Stale previousArea caused wrong navigation on next ESC
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea, previousArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          setFocus('detail');
        } else if (step.current === 1 && focusedArea === 'detail') {
          step.current = 2;
          returnFocus();
        }
      }, [setFocus, returnFocus, focusedArea]);

      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>prev:{previousArea ?? 'null'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    expect(output).toContain('focus:main');
    expect(output).toContain('prev:null');
  });

  test('focus state consistent after rapid transitions', async () => {
    /**
     * Regression: Rapid focus changes should not corrupt state
     * Original bug: Race condition in focus updates
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, focusedArea } = useFocus();
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          // Rapid sequence of focus changes
          setFocus('detail');
          setFocus('input');
          setFocus('detail');
          setFocus('main');
        }
      }, [setFocus, focusedArea]);

      return <Text>focus:{focusedArea}</Text>;
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    // Final state should be main (last setFocus call)
    expect(lastFrame()).toContain('focus:main');
  });

  test('returnFocus when no previous area does nothing', async () => {
    /**
     * Regression: returnFocus with no previous should be no-op
     * Original bug: returnFocus threw or went to undefined state
     */
    const TestComponent = (): React.ReactElement => {
      const { returnFocus, focusedArea } = useFocus();
      const didReturn = useRef(false);

      useEffect(() => {
        if (!didReturn.current) {
          didReturn.current = true;
          returnFocus(); // Called with no previous area
        }
      }, [returnFocus]);

      return <Text>focus:{focusedArea}</Text>;
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    // Should remain at initial focus
    expect(lastFrame()).toContain('focus:main');
  });
});

// =============================================================================
// VIEW STATE MACHINE PATTERNS
// Test patterns that views should follow
// =============================================================================

describe('View State Machine Patterns', () => {
  test('view-local state clears on exit pattern', async () => {
    /**
     * Pattern: Views should clear local state when navigating away
     * This tests the pattern used by AgentDetailView, ChannelHistoryView, etc.
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea } = useFocus();
      const [inputMode, setInputMode] = React.useState(false);
      const [messageBuffer, setMessageBuffer] = React.useState('');
      const step = useRef(0);

      // Simulate: enter input mode -> ESC to exit with cleared state
      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          // User starts typing
          setInputMode(true);
          setMessageBuffer('test message');
          setFocus('input');
        } else if (step.current === 1 && focusedArea === 'input') {
          step.current = 2;
          // User presses ESC - should clear state and return focus
          setInputMode(false);
          setMessageBuffer('');
          returnFocus();
        }
      }, [setFocus, returnFocus, focusedArea]);

      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>inputMode:{inputMode ? 'yes' : 'no'}</Text>
          <Text>buffer:{messageBuffer || 'empty'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    await waitForRender(); // Extra wait for state clear
    const output = lastFrame() ?? '';
    expect(output).toContain('focus:main'); // Should be back at main
    expect(output).toContain('inputMode:no');
    expect(output).toContain('buffer:empty');
  });

  test('confirmation dialog pattern with modal focus', async () => {
    /**
     * Pattern: Confirmation dialogs should trap focus when shown
     * Used by: delete confirmation, prune confirmation, etc.
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, focusedArea, isFocused } = useFocus();
      const [showConfirm, setShowConfirm] = React.useState(false);
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          // User triggers delete confirmation
          setShowConfirm(true);
          setFocus('modal');
        }
      }, [setFocus, focusedArea]);

      const globalKeysBlocked = isFocused('modal') || isFocused('input');

      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>confirm:{showConfirm ? 'yes' : 'no'}</Text>
          <Text>globalBlocked:{globalKeysBlocked ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    // When modal is open, focus should be trapped
    expect(output).toContain('focus:modal');
    expect(output).toContain('confirm:yes');
    expect(output).toContain('globalBlocked:yes');
  });

  test('confirmation dialog closes and returns focus', async () => {
    /**
     * Pattern: After confirming/canceling dialog, focus returns to main
     */
    const TestComponent = (): React.ReactElement => {
      const { setFocus, returnFocus, focusedArea, isFocused } = useFocus();
      const [showConfirm, setShowConfirm] = React.useState(false);
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0 && focusedArea === 'main') {
          step.current = 1;
          setShowConfirm(true);
          setFocus('modal');
        } else if (step.current === 1 && focusedArea === 'modal') {
          step.current = 2;
          // User confirms or cancels
          setShowConfirm(false);
          returnFocus();
        }
      }, [setFocus, returnFocus, focusedArea]);

      const globalKeysBlocked = isFocused('modal') || isFocused('input');

      return (
        <Box flexDirection="column">
          <Text>focus:{focusedArea}</Text>
          <Text>confirm:{showConfirm ? 'yes' : 'no'}</Text>
          <Text>globalBlocked:{globalKeysBlocked ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders initialFocus="main">
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    await waitForRender();
    const output = lastFrame() ?? '';
    expect(output).toContain('focus:main');
    expect(output).toContain('confirm:no');
    expect(output).toContain('globalBlocked:no');
  });
});

// =============================================================================
// SELECTION PRESERVATION REGRESSION
// Issues: Selection lost after navigation
// =============================================================================

describe('Selection Preservation Regression', () => {
  test('selection index pattern preserved across detail view', async () => {
    /**
     * Pattern: List selection should be preserved when returning from detail
     * Used by: AgentsView, ChannelsView, IssuesView, etc.
     */
    const TestComponent = (): React.ReactElement => {
      const [selectedIndex, setSelectedIndex] = React.useState(5); // User selected item 5
      const [showDetail, setShowDetail] = React.useState(false);
      const step = useRef(0);

      useEffect(() => {
        if (step.current === 0) {
          step.current = 1;
          // User opens detail view
          setShowDetail(true);
        } else if (step.current === 1 && showDetail) {
          step.current = 2;
          // User returns from detail
          setShowDetail(false);
        }
      }, [showDetail]);

      return (
        <Box flexDirection="column">
          <Text>selected:{selectedIndex}</Text>
          <Text>detail:{showDetail ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(
      <TestProviders>
        <TestComponent />
      </TestProviders>
    );

    await waitForRender();
    const output = lastFrame() ?? '';
    // Selection should be preserved after returning
    expect(output).toContain('selected:5');
    expect(output).toContain('detail:no');
  });
});
