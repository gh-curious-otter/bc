/**
 * useKeyboardNavigation - Hook for keyboard-based navigation
 */

import { useInput } from 'ink';
import { useNavigation } from './NavigationContext';
import { useFocus } from './FocusContext';

export interface UseKeyboardNavigationOptions {
  /** Disable keyboard handling (useful for testing or when another component captures input) */
  disabled?: boolean;
  /** Custom quit handler (defaults to process.exit(0)) */
  onQuit?: () => void;
}

/**
 * Hook that handles global keyboard navigation
 * - Number keys (1-9) switch tabs
 * - ? shows help
 * - ESC goes back/home
 * - q quits the application
 */
export function useKeyboardNavigation(options: UseKeyboardNavigationOptions = {}): void {
  const { disabled = false, onQuit } = options;
  const { navigate, goHome, getTabByKey } = useNavigation();
  const { isFocused } = useFocus();

  useInput(
    (input, key) => {
      /**
       * Guard against keybinds during text input
       *
       * When a component (like ChannelsView) is in input mode, it calls setFocus('input')
       * to indicate the user is typing a message. This prevents the global keybinds
       * (q to quit, 1-9 for tab switching, ESC for home) from triggering.
       *
       * Returning early here disables ALL global keybinds while the user is composing.
       * When the user finishes typing (presses Enter or Escape), the focus is restored
       * and keybinds are re-enabled.
       *
       * This fixes issue #653: Keybinds not being re-enabled after typing in channels.
       */
      if (isFocused('input')) {
        console.error(`[useKeyboardNavigation] Input focused, blocking keybind: "${input || key.escape ? 'ESC' : 'other'}"`);
        return;
      }

      // Tab navigation with number keys (1-9) and ?
      const tab = getTabByKey(input);
      if (tab) {
        navigate(tab.view);
        return;
      }

      // ESC: go home
      if (key.escape) {
        goHome();
        return;
      }

      // q: quit application
      if (input === 'q') {
        if (onQuit) {
          onQuit();
        } else {
          process.exit(0);
        }
        return;
      }
    },
    { isActive: !disabled }
  );
}

export default useKeyboardNavigation;
