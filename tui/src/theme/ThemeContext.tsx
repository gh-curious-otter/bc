/**
 * ThemeContext - Provides theming support for bc TUI
 *
 * Features:
 * - Auto-detect terminal dark/light mode
 * - Theme provider with context
 * - useTheme hook for accessing colors
 * - Support for custom theme overrides
 */

import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useMemo,
} from 'react';
import type { Theme, ThemeMode, ThemeColors, ThemeConfig } from './types';
import { darkTheme, lightTheme, applyOverrides } from './themes';
import { detectColorScheme } from './detectColorScheme';

interface ThemeContextValue {
  /** Current active theme */
  theme: Theme;
  /** Current theme mode */
  mode: ThemeMode;
  /** Detected terminal color scheme */
  detectedScheme: 'dark' | 'light';
  /** Switch theme mode */
  setMode: (mode: ThemeMode) => void;
  /** Toggle between dark and light */
  toggleTheme: () => void;
  /** Get a specific color from the theme */
  color: <K extends keyof ThemeColors>(key: K) => ThemeColors[K];
  /** Check if current theme is dark */
  isDark: boolean;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

interface ThemeProviderProps {
  children: React.ReactNode;
  /** Initial theme configuration */
  config?: ThemeConfig;
}

/**
 * ThemeProvider - Wrap your app to enable theming
 */
export function ThemeProvider({
  children,
  config,
}: ThemeProviderProps): React.ReactElement {
  const [mode, setModeState] = useState<ThemeMode>(config?.mode ?? 'auto');
  const [detectedScheme] = useState<'dark' | 'light'>(() => detectColorScheme());

  // Determine effective theme based on mode
  const effectiveMode = useMemo((): 'dark' | 'light' => {
    if (mode === 'auto') {
      return detectedScheme;
    }
    return mode;
  }, [mode, detectedScheme]);

  // Build the theme with any overrides
  const theme = useMemo((): Theme => {
    const baseTheme = effectiveMode === 'light' ? lightTheme : darkTheme;
    if (config?.overrides) {
      return applyOverrides(baseTheme, config.overrides);
    }
    return baseTheme;
  }, [effectiveMode, config?.overrides]);

  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode);
  }, []);

  const toggleTheme = useCallback(() => {
    setModeState((current) => {
      if (current === 'auto') {
        // Auto -> opposite of detected
        return detectedScheme === 'dark' ? 'light' : 'dark';
      }
      return current === 'dark' ? 'light' : 'dark';
    });
  }, [detectedScheme]);

  const color = useCallback(
    <K extends keyof ThemeColors>(key: K): ThemeColors[K] => {
      return theme.colors[key];
    },
    [theme]
  );

  const value = useMemo(
    (): ThemeContextValue => ({
      theme,
      mode,
      detectedScheme,
      setMode,
      toggleTheme,
      color,
      isDark: effectiveMode === 'dark',
    }),
    [theme, mode, detectedScheme, setMode, toggleTheme, color, effectiveMode]
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

/**
 * useTheme - Access theme context
 */
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

/**
 * useThemeColor - Get a specific color from the theme
 *
 * Convenience hook for getting a single color value.
 */
export function useThemeColor<K extends keyof ThemeColors>(
  key: K
): ThemeColors[K] {
  const { color } = useTheme();
  return color(key);
}

/**
 * useThemeColors - Get multiple colors from the theme
 *
 * Returns an object with the requested color keys.
 */
export function useThemeColors<K extends keyof ThemeColors>(
  keys: K[]
): Pick<ThemeColors, K> {
  const { theme } = useTheme();
  return useMemo(() => {
    const result = {} as Pick<ThemeColors, K>;
    for (const key of keys) {
      result[key] = theme.colors[key];
    }
    return result;
  }, [theme, keys]);
}

export default ThemeContext;
