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
  type BreadcrumbItem,
} from './NavigationContext';

export { Breadcrumb } from './Breadcrumb';

export {
  useKeyboardNavigation,
  type UseKeyboardNavigationOptions,
} from './useKeyboardNavigation';

export { TabBar, type TabBarProps } from './TabBar';

export { Drawer, type DrawerProps } from './Drawer';

export { TopTabBar, type TopTabBarProps } from './TopTabBar';

export {
  FocusProvider,
  useFocus,
  useIsFocused,
  type FocusArea,
} from './FocusContext';
