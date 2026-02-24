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
  /** Command palette handler (triggered by Ctrl+K) */
  onCommandPalette?: () => void;
}

/**
 * Hook that handles global keyboard navigation
 * - Tab/Shift+Tab cycles views
 * - ? shows help
 * - M goes to Memory view
 * - I goes to Issues view
 * - ESC goes back/home
 * - Ctrl+R refreshes all data
 * - q quits the application
 *
 * Issue #1467: Removed 1-9 number shortcuts.
 * Navigation now uses j/k + Enter in Drawer component.
 * Issue #1686: Added M shortcut for Memory view.
 * Issue #1765: Removed Routing tab (unused static data).
 * Issue #1779: Added I shortcut for Issues view.
 */
export function useKeyboardNavigation(options: UseKeyboardNavigationOptions = {}): void {
  const { disabled = false, onQuit, onRefresh, onCommandPalette } = options;
  const { navigate, goHome, getTabByKey, nextTab, prevTab } = useNavigation();
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
      // Skip ALL global keybinds when user is in input mode (typing)
      if (isFocused('input')) {
        return;
      }

      // Issue #1467: Removed 1-9 number shortcuts
      // Navigation now uses j/k + Enter in Drawer component
      // Global shortcuts: ? (help), M (memory), I (issues)
      if (input === '?') {
        const helpTab = getTabByKey('?');
        if (helpTab) {
          navigate(helpTab.view);
          return;
        }
      }

      // M: go to Memory view (#1686)
      if (input === 'M') {
        const memoryTab = getTabByKey('M');
        if (memoryTab) {
          navigate(memoryTab.view);
          return;
        }
      }

      // I: go to Issues view (#1779)
      if (input === 'I') {
        const issuesTab = getTabByKey('I');
        if (issuesTab) {
          navigate(issuesTab.view);
          return;
        }
      }

      // Tab key: next view, Shift+Tab: previous view
      // Note: j/k are handled by Drawer component for list navigation
      if (key.tab) {
        if (key.shift) {
          prevTab();
        } else {
          nextTab();
        }
        return;
      }

      // ESC: go home (skip when local view handles ESC)
      if (key.escape && !isFocused('view')) {
        goHome();
        return;
      }

      // Ctrl+R: refresh all data
      if (key.ctrl && input === 'r') {
        if (onRefresh) {
          onRefresh();
        }
        return;
      }

      // Ctrl+K: open command palette
      if (key.ctrl && input === 'k') {
        if (onCommandPalette) {
          onCommandPalette();
        }
        return;
      }

      // q: quit application (skip when local view handles q, same pattern as ESC)
      if (input === 'q' && !isFocused('view')) {
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
