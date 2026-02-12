/**
 * useKeyboardNavigation - Hook for keyboard-based navigation
 */

import { useInput } from 'ink';
import { useNavigation } from './NavigationContext';

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

  useInput(
    (input, key) => {
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
