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
  /** Global refresh handler (triggered by Ctrl+R) */
  onRefresh?: () => void;
}

/**
 * Hook that handles global keyboard navigation
 * - Number keys (1-9) switch tabs
 * - Tab/Shift+Tab cycles tabs
 * - ? shows help
 * - ESC goes back/home
 * - Ctrl+R refreshes all data
 * - q quits the application
 */
export function useKeyboardNavigation(options: UseKeyboardNavigationOptions = {}): void {
  const { disabled = false, onQuit, onRefresh } = options;
  const { navigate, goHome, getTabByKey, nextTab, prevTab, breadcrumbs } = useNavigation();
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
        return;
      }

      // Tab navigation with number keys (1-9) and ?
      const tab = getTabByKey(input);
      if (tab) {
        navigate(tab.view);
        return;
      }

      // Tab key: next tab, Shift+Tab: previous tab
      if (key.tab) {
        if (key.shift) {
          prevTab();
        } else {
          nextTab();
        }
        return;
      }

      // ESC: go back if in sub-view (breadcrumbs exist), otherwise go home
      // When viewing channel history, agent details, etc., breadcrumbs are set.
      // ESC should return to the parent view (handled by the component), not dashboard.
      // Only go home if no breadcrumbs exist (already at top level of a tab).
      if (key.escape) {
        if (breadcrumbs.length === 0) {
          goHome();
        }
        // If breadcrumbs exist, let the component handle ESC to go back
        return;
      }

      // Ctrl+R: refresh all data
      if (key.ctrl && input === 'r') {
        if (onRefresh) {
          onRefresh();
        }
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
