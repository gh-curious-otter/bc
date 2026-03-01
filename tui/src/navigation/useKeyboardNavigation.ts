/**
 * useKeyboardNavigation - k9s-style keyboard navigation
 */

import { useInput } from 'ink';
import { useNavigation } from './NavigationContext';
import { useFocus } from './FocusContext';

export interface UseKeyboardNavigationOptions {
  disabled?: boolean;
  onQuit?: () => void;
  onRefresh?: () => void;
  onCommandBar?: () => void;
  onFilterBar?: () => void;
}

/**
 * Hook that handles global keyboard navigation
 * - : → Open command bar (k9s-style primary navigation)
 * - / → Open filter bar
 * - ? → Go to help
 * - Tab/Shift+Tab → Cycle views
 * - Esc → Cancel input / go back / go home
 * - q → Quit
 * - Ctrl+R → Refresh
 */
export function useKeyboardNavigation(options: UseKeyboardNavigationOptions = {}): void {
  const { disabled = false, onQuit, onRefresh, onCommandBar, onFilterBar } = options;
  const { navigate, goHome, nextTab, prevTab } = useNavigation();
  const { isFocused } = useFocus();

  useInput(
    (input, key) => {
      // Skip all global keybinds when in input, command, or filter mode
      if (isFocused('input') || isFocused('command') || isFocused('filter')) {
        return;
      }

      // : → Open command bar (k9s-style)
      if (input === ':') {
        if (onCommandBar) {
          onCommandBar();
        }
        return;
      }

      // / → Open filter bar
      if (input === '/') {
        if (onFilterBar) {
          onFilterBar();
        }
        return;
      }

      // ? → Help
      if (input === '?') {
        navigate('help');
        return;
      }

      // Tab/Shift+Tab → cycle views
      if (key.tab) {
        if (key.shift) {
          prevTab();
        } else {
          nextTab();
        }
        return;
      }

      // ESC → go home (skip when local view handles ESC)
      if (key.escape && !isFocused('view')) {
        goHome();
        return;
      }

      // Ctrl+R → refresh
      if (key.ctrl && input === 'r') {
        if (onRefresh) {
          onRefresh();
        }
        return;
      }

      // q → quit (skip when local view handles q)
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
