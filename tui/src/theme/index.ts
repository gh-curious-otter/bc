/**
 * Theme exports for bc TUI
 *
 * Provides theming support including:
 * - Auto-detection of terminal dark/light mode
 * - ThemeProvider for React context
 * - useTheme hook for accessing colors
 * - Pre-defined dark and light themes
 */

// Types
export type {
  Theme,
  ThemeColors,
  ThemeMode,
  ThemeConfig,
  TerminalColor,
} from './types';

// Context and hooks
export {
  ThemeProvider,
  useTheme,
  useThemeColor,
  useThemeColors,
} from './ThemeContext';

// Themes
export { darkTheme, lightTheme, getTheme, applyOverrides } from './themes';

// Detection utilities
export {
  detectColorScheme,
  supportsExtendedColors,
  supportsColors,
} from './detectColorScheme';
