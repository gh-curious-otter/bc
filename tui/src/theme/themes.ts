/**
 * Default theme definitions for bc TUI
 */

import type { Theme, ThemeColors } from './types';

export const darkTheme: Theme = {
  name: 'dark',
  mode: 'dark',
  colors: {
    primary: 'cyan',
    secondary: 'blue',
    accent: 'magenta',
    text: 'white',
    textMuted: 'gray',
    textInverse: 'black',
    success: 'green',
    warning: 'yellow',
    error: 'red',
    info: 'cyan',
    agentIdle: 'gray',
    agentWorking: 'blue',
    agentDone: 'green',
    agentStuck: 'yellow',
    agentError: 'red',
    border: 'gray',
    borderFocused: 'cyan',
    selection: 'cyan',
    highlight: 'yellow',
    headerTitle: 'cyan',
    footerText: 'gray',
    badge: 'magenta',
  },
};

export const lightTheme: Theme = {
  name: 'light',
  mode: 'light',
  colors: {
    primary: 'blue',
    secondary: 'cyan',
    accent: 'magenta',
    text: 'black',
    textMuted: '#666666', // 5.7:1 on white — WCAG AA compliant
    textInverse: 'white',
    success: 'green',
    warning: 'yellow',
    error: 'red',
    info: 'blue',
    agentIdle: '#767676', // 4.5:1 on white — WCAG AA compliant
    agentWorking: 'blue',
    agentDone: 'green',
    agentStuck: 'yellow',
    agentError: 'red',
    border: '#767676', // 4.5:1 on white — exceeds 3:1 AA for UI elements
    borderFocused: 'blue',
    selection: 'blue',
    highlight: 'yellow',
    headerTitle: 'blue',
    footerText: '#666666', // 5.7:1 on white — WCAG AA compliant
    badge: 'magenta',
  },
};

export type ThemeName = 'dark' | 'light';

export const themes: Record<ThemeName, Theme> = {
  dark: darkTheme,
  light: lightTheme,
};

export function getTheme(name: ThemeName): Theme {
  return themes[name];
}

export function applyOverrides(
  theme: Theme,
  overrides: Partial<ThemeColors>
): Theme {
  return {
    ...theme,
    colors: {
      ...theme.colors,
      ...overrides,
    },
  };
}
