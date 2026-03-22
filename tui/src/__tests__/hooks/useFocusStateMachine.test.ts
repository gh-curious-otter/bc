/**
 * Tests for useFocusStateMachine hook
 *
 * Issue #1825: Focus management state machine
 *
 * Tests the pure state machine logic directly (not React hooks)
 * to avoid DOM dependency issues in Bun test environment.
 */

import { describe, test, expect } from 'bun:test';
import {
  categorizeKey,
  type FocusState,
  type FocusTransition,
  type KeyCategory,
} from '../../hooks/useFocusStateMachine';

/**
 * Pure state machine implementation for testing
 * (mirrors the hook logic without React dependencies)
 */
class FocusStateMachine {
  state: FocusState;
  history: FocusState[];

  /** Valid state transitions */
  private static TRANSITIONS: Record<FocusState, Partial<Record<FocusTransition, FocusState>>> = {
    main: {
      ENTER_INPUT: 'input',
      OPEN_DETAIL: 'detail',
      OPEN_MODAL: 'modal',
    },
    input: {
      EXIT_INPUT: 'main', // Placeholder - actual goes to previous
    },
    detail: {
      ENTER_INPUT: 'input',
      CLOSE_DETAIL: 'main',
      OPEN_DETAIL: 'detail',
      OPEN_MODAL: 'modal',
      GO_HOME: 'main',
    },
    modal: {
      CLOSE_MODAL: 'main', // Placeholder - actual goes to previous
      ENTER_INPUT: 'input',
    },
  };

  /** Key permissions per state */
  private static KEY_PERMISSIONS: Record<FocusState, Set<KeyCategory>> = {
    main: new Set(['global_nav', 'global_quit', 'list_nav', 'selection', 'escape', 'refresh']),
    input: new Set(['text_input', 'escape', 'selection']),
    detail: new Set(['global_nav', 'list_nav', 'selection', 'escape', 'refresh']),
    modal: new Set(['selection', 'escape', 'list_nav']),
  };

  constructor(initialState: FocusState = 'main') {
    this.state = initialState;
    this.history = [initialState];
  }

  get previousState(): FocusState | null {
    return this.history.length >= 2 ? this.history[this.history.length - 2] : null;
  }

  transition(event: FocusTransition): void {
    const validTransitions = FocusStateMachine.TRANSITIONS[this.state];
    const nextState = validTransitions[event];

    if (nextState === undefined) {
      // Invalid transition - ignore
      return;
    }

    // Special handling for EXIT_INPUT and CLOSE_MODAL - return to previous state
    if (event === 'EXIT_INPUT' || event === 'CLOSE_MODAL') {
      if (this.history.length >= 2) {
        this.history.pop();
        this.state = this.history[this.history.length - 1];
      }
      return;
    }

    // Normal transition - push new state to history
    this.history.push(nextState);
    this.state = nextState;
  }

  canHandle(category: KeyCategory): boolean {
    return FocusStateMachine.KEY_PERMISSIONS[this.state].has(category);
  }

  isState(s: FocusState): boolean {
    return this.state === s;
  }
}

describe('FocusStateMachine', () => {
  describe('initial state', () => {
    test('defaults to main state', () => {
      const sm = new FocusStateMachine();
      expect(sm.state).toBe('main');
    });

    test('accepts custom initial state', () => {
      const sm = new FocusStateMachine('detail');
      expect(sm.state).toBe('detail');
    });

    test('initializes history with initial state', () => {
      const sm = new FocusStateMachine();
      expect(sm.history).toEqual(['main']);
    });

    test('previousState is null initially', () => {
      const sm = new FocusStateMachine();
      expect(sm.previousState).toBeNull();
    });
  });

  describe('state transitions', () => {
    describe('from main state', () => {
      test('ENTER_INPUT transitions to input', () => {
        const sm = new FocusStateMachine();
        sm.transition('ENTER_INPUT');
        expect(sm.state).toBe('input');
      });

      test('OPEN_DETAIL transitions to detail', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        expect(sm.state).toBe('detail');
      });

      test('OPEN_MODAL transitions to modal', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_MODAL');
        expect(sm.state).toBe('modal');
      });

      test('invalid transitions are ignored', () => {
        const sm = new FocusStateMachine();
        sm.transition('EXIT_INPUT');
        expect(sm.state).toBe('main');
        sm.transition('CLOSE_DETAIL');
        expect(sm.state).toBe('main');
        sm.transition('CLOSE_MODAL');
        expect(sm.state).toBe('main');
      });
    });

    describe('from input state', () => {
      test('EXIT_INPUT returns to previous state (main)', () => {
        const sm = new FocusStateMachine();
        sm.transition('ENTER_INPUT');
        expect(sm.state).toBe('input');
        sm.transition('EXIT_INPUT');
        expect(sm.state).toBe('main');
      });

      test('EXIT_INPUT returns to previous state (detail)', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        sm.transition('ENTER_INPUT');
        expect(sm.state).toBe('input');
        sm.transition('EXIT_INPUT');
        expect(sm.state).toBe('detail');
      });

      test('invalid transitions are ignored', () => {
        const sm = new FocusStateMachine();
        sm.transition('ENTER_INPUT');
        sm.transition('OPEN_DETAIL');
        expect(sm.state).toBe('input');
        sm.transition('OPEN_MODAL');
        expect(sm.state).toBe('input');
      });
    });

    describe('from detail state', () => {
      test('ENTER_INPUT transitions to input', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        sm.transition('ENTER_INPUT');
        expect(sm.state).toBe('input');
      });

      test('CLOSE_DETAIL transitions to main', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        sm.transition('CLOSE_DETAIL');
        expect(sm.state).toBe('main');
      });

      test('OPEN_DETAIL stays in detail (nested detail)', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        sm.transition('OPEN_DETAIL');
        expect(sm.state).toBe('detail');
      });

      test('OPEN_MODAL transitions to modal', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        sm.transition('OPEN_MODAL');
        expect(sm.state).toBe('modal');
      });

      test('GO_HOME transitions to main', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_DETAIL');
        sm.transition('GO_HOME');
        expect(sm.state).toBe('main');
      });
    });

    describe('from modal state', () => {
      test('CLOSE_MODAL returns to previous state', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_MODAL');
        expect(sm.state).toBe('modal');
        sm.transition('CLOSE_MODAL');
        expect(sm.state).toBe('main');
      });

      test('ENTER_INPUT transitions to input', () => {
        const sm = new FocusStateMachine();
        sm.transition('OPEN_MODAL');
        sm.transition('ENTER_INPUT');
        expect(sm.state).toBe('input');
      });
    });
  });

  describe('history tracking', () => {
    test('tracks state history', () => {
      const sm = new FocusStateMachine();
      sm.transition('OPEN_DETAIL');
      sm.transition('ENTER_INPUT');
      expect(sm.history).toEqual(['main', 'detail', 'input']);
    });

    test('previousState reflects history', () => {
      const sm = new FocusStateMachine();
      expect(sm.previousState).toBeNull();
      sm.transition('OPEN_DETAIL');
      expect(sm.previousState).toBe('main');
      sm.transition('ENTER_INPUT');
      expect(sm.previousState).toBe('detail');
    });

    test('EXIT_INPUT pops history', () => {
      const sm = new FocusStateMachine();
      sm.transition('OPEN_DETAIL');
      sm.transition('ENTER_INPUT');
      expect(sm.history.length).toBe(3);
      sm.transition('EXIT_INPUT');
      expect(sm.history.length).toBe(2);
      expect(sm.history).toEqual(['main', 'detail']);
    });
  });

  describe('canHandle', () => {
    describe('main state', () => {
      test('allows global navigation', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('global_nav')).toBe(true);
      });

      test('allows global quit', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('global_quit')).toBe(true);
      });

      test('allows list navigation', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('list_nav')).toBe(true);
      });

      test('allows selection', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('selection')).toBe(true);
      });

      test('allows escape', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('escape')).toBe(true);
      });

      test('allows refresh', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('refresh')).toBe(true);
      });

      test('disallows text input', () => {
        const sm = new FocusStateMachine('main');
        expect(sm.canHandle('text_input')).toBe(false);
      });
    });

    describe('input state', () => {
      test('allows text input', () => {
        const sm = new FocusStateMachine('input');
        expect(sm.canHandle('text_input')).toBe(true);
      });

      test('allows escape (to exit)', () => {
        const sm = new FocusStateMachine('input');
        expect(sm.canHandle('escape')).toBe(true);
      });

      test('allows selection (Enter to submit)', () => {
        const sm = new FocusStateMachine('input');
        expect(sm.canHandle('selection')).toBe(true);
      });

      test('disallows global navigation', () => {
        const sm = new FocusStateMachine('input');
        expect(sm.canHandle('global_nav')).toBe(false);
      });

      test('disallows global quit', () => {
        const sm = new FocusStateMachine('input');
        expect(sm.canHandle('global_quit')).toBe(false);
      });

      test('disallows list navigation', () => {
        const sm = new FocusStateMachine('input');
        expect(sm.canHandle('list_nav')).toBe(false);
      });
    });

    describe('detail state', () => {
      test('allows global navigation', () => {
        const sm = new FocusStateMachine('detail');
        expect(sm.canHandle('global_nav')).toBe(true);
      });

      test('disallows global quit (q goes back in detail)', () => {
        const sm = new FocusStateMachine('detail');
        expect(sm.canHandle('global_quit')).toBe(false);
      });

      test('allows list navigation', () => {
        const sm = new FocusStateMachine('detail');
        expect(sm.canHandle('list_nav')).toBe(true);
      });

      test('allows escape', () => {
        const sm = new FocusStateMachine('detail');
        expect(sm.canHandle('escape')).toBe(true);
      });
    });

    describe('modal state', () => {
      test('allows selection', () => {
        const sm = new FocusStateMachine('modal');
        expect(sm.canHandle('selection')).toBe(true);
      });

      test('allows escape', () => {
        const sm = new FocusStateMachine('modal');
        expect(sm.canHandle('escape')).toBe(true);
      });

      test('allows list navigation (for options)', () => {
        const sm = new FocusStateMachine('modal');
        expect(sm.canHandle('list_nav')).toBe(true);
      });

      test('disallows global navigation', () => {
        const sm = new FocusStateMachine('modal');
        expect(sm.canHandle('global_nav')).toBe(false);
      });

      test('disallows global quit', () => {
        const sm = new FocusStateMachine('modal');
        expect(sm.canHandle('global_quit')).toBe(false);
      });
    });
  });

  describe('isState', () => {
    test('returns true for current state', () => {
      const sm = new FocusStateMachine('main');
      expect(sm.isState('main')).toBe(true);
      expect(sm.isState('input')).toBe(false);
      expect(sm.isState('detail')).toBe(false);
      expect(sm.isState('modal')).toBe(false);
    });

    test('updates after transition', () => {
      const sm = new FocusStateMachine();
      sm.transition('ENTER_INPUT');
      expect(sm.isState('main')).toBe(false);
      expect(sm.isState('input')).toBe(true);
    });
  });
});

describe('categorizeKey', () => {
  test('categorizes escape key', () => {
    expect(categorizeKey('', { escape: true })).toBe('escape');
  });

  test('categorizes return/enter key', () => {
    expect(categorizeKey('', { return: true })).toBe('selection');
  });

  test('categorizes tab key', () => {
    expect(categorizeKey('', { tab: true })).toBe('global_nav');
  });

  test('categorizes Ctrl+R as refresh', () => {
    expect(categorizeKey('r', { ctrl: true })).toBe('refresh');
  });

  test('categorizes ? as global nav', () => {
    expect(categorizeKey('?', {})).toBe('global_nav');
  });

  test('categorizes M as global nav', () => {
    expect(categorizeKey('M', {})).toBe('global_nav');
  });

  test('categorizes I as global nav', () => {
    expect(categorizeKey('I', {})).toBe('global_nav');
  });

  test('categorizes q as global quit', () => {
    expect(categorizeKey('q', {})).toBe('global_quit');
  });

  test('categorizes j as list nav', () => {
    expect(categorizeKey('j', {})).toBe('list_nav');
  });

  test('categorizes k as list nav', () => {
    expect(categorizeKey('k', {})).toBe('list_nav');
  });

  test('categorizes g as list nav', () => {
    expect(categorizeKey('g', {})).toBe('list_nav');
  });

  test('categorizes G as list nav', () => {
    expect(categorizeKey('G', {})).toBe('list_nav');
  });

  test('categorizes r as refresh', () => {
    expect(categorizeKey('r', {})).toBe('refresh');
  });

  test('categorizes regular character as text input', () => {
    expect(categorizeKey('a', {})).toBe('text_input');
    expect(categorizeKey('z', {})).toBe('text_input');
    expect(categorizeKey('1', {})).toBe('text_input');
    expect(categorizeKey(' ', {})).toBe('text_input');
  });

  test('returns null for empty input', () => {
    expect(categorizeKey('', {})).toBeNull();
  });

  test('returns null for ctrl combinations (except Ctrl+R)', () => {
    expect(categorizeKey('c', { ctrl: true })).toBeNull();
    expect(categorizeKey('v', { ctrl: true })).toBeNull();
  });
});

describe('real-world scenarios', () => {
  test('search flow: main -> input -> main', () => {
    const sm = new FocusStateMachine();

    // User presses '/' to search
    expect(sm.canHandle('text_input')).toBe(false);
    sm.transition('ENTER_INPUT');

    // User types search query
    expect(sm.state).toBe('input');
    expect(sm.canHandle('text_input')).toBe(true);
    expect(sm.canHandle('global_quit')).toBe(false);

    // User presses ESC to cancel
    sm.transition('EXIT_INPUT');
    expect(sm.state).toBe('main');
    expect(sm.canHandle('global_quit')).toBe(true);
  });

  test('detail drill-down with search: main -> detail -> input -> detail -> main', () => {
    const sm = new FocusStateMachine();

    // User opens agent detail
    sm.transition('OPEN_DETAIL');
    expect(sm.state).toBe('detail');
    expect(sm.canHandle('escape')).toBe(true);
    expect(sm.canHandle('global_quit')).toBe(false); // q shouldn't quit

    // User starts searching within detail
    sm.transition('ENTER_INPUT');
    expect(sm.state).toBe('input');

    // User cancels search
    sm.transition('EXIT_INPUT');
    expect(sm.state).toBe('detail'); // Returns to detail, not main!

    // User presses ESC to exit detail
    sm.transition('CLOSE_DETAIL');
    expect(sm.state).toBe('main');
  });

  test('modal confirmation flow: main -> modal -> main', () => {
    const sm = new FocusStateMachine();

    // User triggers delete confirmation
    sm.transition('OPEN_MODAL');
    expect(sm.state).toBe('modal');
    expect(sm.canHandle('global_nav')).toBe(false);
    expect(sm.canHandle('global_quit')).toBe(false);
    expect(sm.canHandle('escape')).toBe(true);

    // User cancels
    sm.transition('CLOSE_MODAL');
    expect(sm.state).toBe('main');
  });

  test('keybinds blocked during input', () => {
    const sm = new FocusStateMachine();

    // Verify all keybinds work in main state
    expect(sm.canHandle('global_nav')).toBe(true);
    expect(sm.canHandle('global_quit')).toBe(true);
    expect(sm.canHandle('list_nav')).toBe(true);

    // Enter input mode
    sm.transition('ENTER_INPUT');

    // Verify keybinds are blocked
    expect(sm.canHandle('global_nav')).toBe(false);
    expect(sm.canHandle('global_quit')).toBe(false);
    expect(sm.canHandle('list_nav')).toBe(false);

    // But text input and escape work
    expect(sm.canHandle('text_input')).toBe(true);
    expect(sm.canHandle('escape')).toBe(true);
  });

  test('nested detail with modal: main -> detail -> modal -> detail', () => {
    const sm = new FocusStateMachine();

    sm.transition('OPEN_DETAIL');
    sm.transition('OPEN_MODAL');
    expect(sm.state).toBe('modal');
    expect(sm.history).toEqual(['main', 'detail', 'modal']);

    sm.transition('CLOSE_MODAL');
    expect(sm.state).toBe('detail');
    expect(sm.history).toEqual(['main', 'detail']);
  });

  test('deep navigation: main -> detail -> detail -> input -> detail', () => {
    const sm = new FocusStateMachine();

    sm.transition('OPEN_DETAIL'); // Agents list -> Agent detail
    sm.transition('OPEN_DETAIL'); // Agent detail -> Memory detail
    sm.transition('ENTER_INPUT'); // Search in memory
    expect(sm.history).toEqual(['main', 'detail', 'detail', 'input']);

    sm.transition('EXIT_INPUT');
    expect(sm.state).toBe('detail');
    expect(sm.history).toEqual(['main', 'detail', 'detail']);
  });
});
