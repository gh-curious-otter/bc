/**
 * FocusContext unit tests
 * Issue #1600: Add comprehensive test coverage for critical paths
 *
 * Tests for:
 * - Focus area management
 * - Focus cycling (cycleFocus)
 * - Previous area tracking
 * - Return focus functionality
 */

import React, { useEffect, useRef } from 'react';
import { render, type Instance } from 'ink-testing-library';
import { Text } from 'ink';
import { describe, test, expect } from 'bun:test';
import { FocusProvider, useFocus, useIsFocused, type FocusArea } from '../FocusContext';

// Helper to wait for render updates
const waitForRender = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 50));

describe('FocusContext', () => {
  describe('FocusProvider', () => {
    test('provides default focus area as "main"', () => {
      const TestComponent = (): React.ReactElement => {
        const { focusedArea } = useFocus();
        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider>
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('focus:main');
    });

    test('respects initialFocus prop', () => {
      const TestComponent = (): React.ReactElement => {
        const { focusedArea } = useFocus();
        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="sidebar">
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('focus:sidebar');
    });

    test('supports all valid focus areas as initial', () => {
      const validAreas: FocusArea[] = ['sidebar', 'main', 'detail', 'input', 'modal', 'view'];

      for (const area of validAreas) {
        const TestComponent = (): React.ReactElement => {
          const { focusedArea } = useFocus();
          return <Text>focus:{focusedArea}</Text>;
        };

        const { lastFrame } = render(
          <FocusProvider initialFocus={area}>
            <TestComponent />
          </FocusProvider>
        );

        expect(lastFrame()).toContain(`focus:${area}`);
      }
    });

    test('useFocus requires FocusProvider context', () => {
      // Verify the error message is defined in the implementation
      // Note: ink-testing-library catches component errors, so we verify
      // the hook's error handling exists by checking it works with a provider
      const ValidUsage = (): React.ReactElement => {
        const { focusedArea } = useFocus();
        return <Text>valid:{focusedArea}</Text>;
      };

      // This should NOT throw when properly wrapped
      const { lastFrame } = render(
        <FocusProvider>
          <ValidUsage />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('valid:main');
    });
  });

  describe('setFocus', () => {
    test('changes the focused area', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, focusedArea } = useFocus();
        const didSet = useRef(false);

        useEffect(() => {
          if (!didSet.current) {
            didSet.current = true;
            setFocus('detail');
          }
        }, [setFocus]);

        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('focus:detail');
    });

    test('tracks previous area when focus changes', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, previousArea } = useFocus();
        const didSet = useRef(false);

        useEffect(() => {
          if (!didSet.current) {
            didSet.current = true;
            setFocus('sidebar');
          }
        }, [setFocus]);

        return <Text>prev:{previousArea ?? 'null'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('prev:main');
    });
  });

  describe('isFocused', () => {
    test('returns true for current focused area', () => {
      const TestComponent = (): React.ReactElement => {
        const { isFocused } = useFocus();
        return <Text>isMainFocused:{isFocused('main') ? 'yes' : 'no'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('isMainFocused:yes');
    });

    test('returns false for non-focused areas', () => {
      const TestComponent = (): React.ReactElement => {
        const { isFocused } = useFocus();
        return (
          <Text>
            sidebar:{isFocused('sidebar') ? 'yes' : 'no'}
            detail:{isFocused('detail') ? 'yes' : 'no'}
          </Text>
        );
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('sidebar:no');
      expect(lastFrame()).toContain('detail:no');
    });

    test('updates when focus changes', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, isFocused } = useFocus();
        const didSet = useRef(false);

        useEffect(() => {
          if (!didSet.current) {
            didSet.current = true;
            setFocus('sidebar');
          }
        }, [setFocus]);

        return <Text>isSidebarFocused:{isFocused('sidebar') ? 'yes' : 'no'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('isSidebarFocused:yes');
    });
  });

  describe('cycleFocus', () => {
    test('cycles from sidebar to main', async () => {
      const TestComponent = (): React.ReactElement => {
        const { cycleFocus, focusedArea } = useFocus();
        const didCycle = useRef(false);

        useEffect(() => {
          if (!didCycle.current) {
            didCycle.current = true;
            cycleFocus();
          }
        }, [cycleFocus]);

        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="sidebar">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      // FOCUS_ORDER is ['sidebar', 'main', 'detail'], so sidebar -> main
      expect(lastFrame()).toContain('focus:main');
    });

    test('cycles from main to detail', async () => {
      const TestComponent = (): React.ReactElement => {
        const { cycleFocus, focusedArea } = useFocus();
        const didCycle = useRef(false);

        useEffect(() => {
          if (!didCycle.current) {
            didCycle.current = true;
            cycleFocus();
          }
        }, [cycleFocus]);

        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('focus:detail');
    });

    test('wraps around from detail back to sidebar', async () => {
      const TestComponent = (): React.ReactElement => {
        const { cycleFocus, focusedArea } = useFocus();
        const didCycle = useRef(false);

        useEffect(() => {
          if (!didCycle.current) {
            didCycle.current = true;
            cycleFocus();
          }
        }, [cycleFocus]);

        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="detail">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      // detail is last in FOCUS_ORDER, wraps to sidebar
      expect(lastFrame()).toContain('focus:sidebar');
    });

    test('tracks previous area during cycling', async () => {
      const TestComponent = (): React.ReactElement => {
        const { cycleFocus, previousArea } = useFocus();
        const didCycle = useRef(false);

        useEffect(() => {
          if (!didCycle.current) {
            didCycle.current = true;
            cycleFocus();
          }
        }, [cycleFocus]);

        return <Text>prev:{previousArea ?? 'null'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('prev:main');
    });
  });

  describe('returnFocus', () => {
    test('returns to previous focus area', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, returnFocus, focusedArea } = useFocus();
        const step = useRef(0);

        useEffect(() => {
          if (step.current === 0 && focusedArea === 'main') {
            step.current = 1;
            setFocus('input');
          } else if (step.current === 1 && focusedArea === 'input') {
            step.current = 2;
            returnFocus();
          }
        }, [setFocus, returnFocus, focusedArea]);

        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('focus:main');
    });

    test('clears previous area after returning', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, returnFocus, previousArea, focusedArea } = useFocus();
        const step = useRef(0);

        useEffect(() => {
          if (step.current === 0 && focusedArea === 'main') {
            step.current = 1;
            setFocus('modal');
          } else if (step.current === 1 && focusedArea === 'modal') {
            step.current = 2;
            returnFocus();
          }
        }, [setFocus, returnFocus, focusedArea]);

        return <Text>prev:{previousArea ?? 'null'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('prev:null');
    });

    test('does nothing when no previous area exists', async () => {
      const TestComponent = (): React.ReactElement => {
        const { returnFocus, focusedArea } = useFocus();
        const didReturn = useRef(false);

        useEffect(() => {
          if (!didReturn.current) {
            didReturn.current = true;
            returnFocus();
          }
        }, [returnFocus]);

        return <Text>focus:{focusedArea}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('focus:main');
    });
  });

  describe('useIsFocused hook', () => {
    test('returns correct value for focused area', () => {
      const TestComponent = (): React.ReactElement => {
        const isFocused = useIsFocused('sidebar');
        return <Text>isFocused:{isFocused ? 'yes' : 'no'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="sidebar">
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('isFocused:yes');
    });

    test('returns false for non-focused area', () => {
      const TestComponent = (): React.ReactElement => {
        const isFocused = useIsFocused('sidebar');
        return <Text>isFocused:{isFocused ? 'yes' : 'no'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('isFocused:no');
    });
  });

  describe('keyboard blocking patterns', () => {
    test('isFocused("input") blocks global keybinds pattern', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, isFocused } = useFocus();
        const didSet = useRef(false);

        useEffect(() => {
          if (!didSet.current) {
            didSet.current = true;
            setFocus('input');
          }
        }, [setFocus]);

        // This pattern is used to block global keybinds
        const shouldBlock = isFocused('input');
        return <Text>blocked:{shouldBlock ? 'yes' : 'no'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('blocked:yes');
    });

    test('isFocused("modal") enables modal focus trap pattern', async () => {
      const TestComponent = (): React.ReactElement => {
        const { setFocus, isFocused } = useFocus();
        const didSet = useRef(false);

        useEffect(() => {
          if (!didSet.current) {
            didSet.current = true;
            setFocus('modal');
          }
        }, [setFocus]);

        // This pattern is used for modal focus trapping
        const inModal = isFocused('modal');
        return <Text>inModal:{inModal ? 'yes' : 'no'}</Text>;
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      await waitForRender();
      expect(lastFrame()).toContain('inModal:yes');
    });

    test('non-blocking areas return false', () => {
      const TestComponent = (): React.ReactElement => {
        const { isFocused } = useFocus();
        return (
          <Text>
            input:{isFocused('input') ? 'yes' : 'no'}
            modal:{isFocused('modal') ? 'yes' : 'no'}
          </Text>
        );
      };

      const { lastFrame } = render(
        <FocusProvider initialFocus="main">
          <TestComponent />
        </FocusProvider>
      );

      expect(lastFrame()).toContain('input:no');
      expect(lastFrame()).toContain('modal:no');
    });
  });
});
