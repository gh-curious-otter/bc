/**
 * Navigation module - k9s-style command navigation
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

export { useKeyboardNavigation, type UseKeyboardNavigationOptions } from './useKeyboardNavigation';

export { FocusProvider, useFocus, useIsFocused, type FocusArea } from './FocusContext';
