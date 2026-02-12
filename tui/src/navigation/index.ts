/**
 * Navigation module - Tab-based navigation framework
 */

export {
  NavigationProvider,
  useNavigation,
  useIsActiveView,
  DEFAULT_TABS,
  type View,
  type TabConfig,
  type NavigationState,
  type NavigationContextValue,
  type NavigationProviderProps,
} from './NavigationContext';

export {
  useKeyboardNavigation,
  type UseKeyboardNavigationOptions,
} from './useKeyboardNavigation';

export { TabBar, type TabBarProps } from './TabBar';
