/**
 * Theme type definitions for bc TUI
 *
 * Defines semantic color mappings for consistent theming across components.
 */

/**
 * Base color palette - terminal-safe colors
 */
export type TerminalColor =
  | 'black'
  | 'red'
  | 'green'
  | 'yellow'
  | 'blue'
  | 'magenta'
  | 'cyan'
  | 'white'
  | 'gray'
  | 'grey'
  | 'blackBright'
  | 'redBright'
  | 'greenBright'
  | 'yellowBright'
  | 'blueBright'
  | 'magentaBright'
  | 'cyanBright'
  | 'whiteBright'
  | `#${string}`;

/**
 * Semantic color definitions for UI elements
 */
export interface ThemeColors {
  // Primary colors
  primary: TerminalColor;
  secondary: TerminalColor;
  accent: TerminalColor;

  // Text colors
  text: TerminalColor;
  textMuted: TerminalColor;
  textInverse: TerminalColor;

  // Status colors
  success: TerminalColor;
  warning: TerminalColor;
  error: TerminalColor;
  info: TerminalColor;

  // Agent state colors
  agentIdle: TerminalColor;
  agentWorking: TerminalColor;
  agentDone: TerminalColor;
  agentStuck: TerminalColor;
  agentError: TerminalColor;

  // UI element colors
  border: TerminalColor;
  borderFocused: TerminalColor;
  selection: TerminalColor;
  highlight: TerminalColor;

  // Component-specific
  headerTitle: TerminalColor;
  footerText: TerminalColor;
  badge: TerminalColor;
}

/**
 * Theme mode
 */
export type ThemeMode = 'dark' | 'light' | 'auto';

/**
 * Complete theme definition
 */
export interface Theme {
  name: string;
  mode: 'dark' | 'light';
  colors: ThemeColors;
}

/**
 * Theme configuration options
 */
export interface ThemeConfig {
  /** Preferred theme mode */
  mode: ThemeMode;
  /** Custom theme overrides */
  overrides?: Partial<ThemeColors>;
}
