/**
 * Default theme definitions for bc TUI
 */

import type { Theme, ThemeColors } from './types';

export const darkTheme: Theme = {
  name: 'dark',
  mode: 'dark',
  colors: {
    primary: '#EA580C',
    secondary: 'blue',
    accent: '#FB923C',
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
    borderFocused: '#EA580C',
    selection: '#EA580C',
    highlight: '#FDBA74',
    headerTitle: '#EA580C',
    footerText: 'gray',
    badge: '#FB923C',
  },
};

export const lightTheme: Theme = {
  name: 'light',
  mode: 'light',
  colors: {
    primary: '#C2410C',
    secondary: 'cyan',
    accent: '#EA580C',
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
    borderFocused: '#C2410C',
    selection: '#C2410C',
    highlight: '#FB923C',
    headerTitle: '#C2410C',
    footerText: '#666666', // 5.7:1 on white — WCAG AA compliant
    badge: '#EA580C',
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

export function applyOverrides(theme: Theme, overrides: Partial<ThemeColors>): Theme {
  return {
    ...theme,
    colors: {
      ...theme.colors,
      ...overrides,
    },
  };
}
