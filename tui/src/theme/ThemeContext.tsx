/**
 * ThemeContext - Provides theming support for bc TUI
 */

import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useMemo,
} from 'react';
import type { Theme, ThemeMode, ThemeColors, ThemeConfig } from './types';
import { darkTheme, lightTheme, applyOverrides, getTheme } from './themes';
import type { ThemeName } from './themes';
import { detectColorScheme } from './detectColorScheme';

interface ThemeContextValue {
  theme: Theme;
  mode: ThemeMode;
  themeName: ThemeName;
  availableThemes: ThemeName[];
  detectedScheme: 'dark' | 'light';
  setMode: (mode: ThemeMode) => void;
  setThemeName: (name: ThemeName) => void;
  cycleTheme: () => void;
  toggleTheme: () => void;
  color: <K extends keyof ThemeColors>(key: K) => ThemeColors[K];
  isDark: boolean;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

interface ThemeProviderProps {
  children: React.ReactNode;
  config?: ThemeConfig;
}

const availableThemes: ThemeName[] = ['dark', 'light'];

export function ThemeProvider({
  children,
  config,
}: ThemeProviderProps): React.ReactElement {
  const [mode, setModeState] = useState<ThemeMode>(config?.mode ?? 'auto');
  const [themeName, setThemeNameState] = useState<ThemeName>('dark');
  const [detectedScheme] = useState<'dark' | 'light'>(() => detectColorScheme());

  const effectiveMode = useMemo((): 'dark' | 'light' => {
    if (mode === 'auto') {
      return detectedScheme;
    }
    return mode;
  }, [mode, detectedScheme]);

  const theme = useMemo((): Theme => {
    const baseTheme = effectiveMode === 'light' ? lightTheme : darkTheme;
    if (config?.overrides) {
      return applyOverrides(baseTheme, config.overrides);
    }
    return baseTheme;
  }, [effectiveMode, config?.overrides]);

  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode);
    if (newMode === 'dark') setThemeNameState('dark');
    if (newMode === 'light') setThemeNameState('light');
  }, []);

  const setThemeName = useCallback((name: ThemeName) => {
    setThemeNameState(name);
    const selectedTheme = getTheme(name);
    setModeState(selectedTheme.mode);
  }, []);

  const cycleTheme = useCallback(() => {
    setThemeNameState((current) => {
      const currentIndex = availableThemes.indexOf(current);
      const nextIndex = (currentIndex + 1) % availableThemes.length;
      const nextTheme = availableThemes[nextIndex];
      const selectedTheme = getTheme(nextTheme);
      setModeState(selectedTheme.mode);
      return nextTheme;
    });
  }, []);

  const toggleTheme = useCallback(() => {
    setModeState((current) => {
      if (current === 'auto') {
        return detectedScheme === 'dark' ? 'light' : 'dark';
      }
      return current === 'dark' ? 'light' : 'dark';
    });
    setThemeNameState((current) => {
      if (current === 'dark') return 'light';
      return 'dark';
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
      themeName,
      availableThemes,
      detectedScheme,
      setMode,
      setThemeName,
      cycleTheme,
      toggleTheme,
      color,
      isDark: theme.mode === 'dark',
    }),
    [theme, mode, themeName, detectedScheme, setMode, setThemeName, cycleTheme, toggleTheme, color]
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

export function useThemeColor<K extends keyof ThemeColors>(
  key: K
): ThemeColors[K] {
  const { color } = useTheme();
  return color(key);
}

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
