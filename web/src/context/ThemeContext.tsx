import { createContext, useCallback, useContext, useEffect, useState } from 'react';

export type ThemeMode = 'dark' | 'light' | 'system';

interface ThemeContextValue {
  mode: ThemeMode;
  resolved: 'dark' | 'light';
  toggle: () => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  mode: 'dark',
  resolved: 'dark',
  toggle: () => {},
});

const STORAGE_KEY = 'bc-theme';
const CYCLE: ThemeMode[] = ['dark', 'light', 'system'];

function getSystemPreference(): 'dark' | 'light' {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function resolve(mode: ThemeMode): 'dark' | 'light' {
  return mode === 'system' ? getSystemPreference() : mode;
}

function readStored(): ThemeMode {
  try {
    const val = localStorage.getItem(STORAGE_KEY);
    if (val === 'dark' || val === 'light' || val === 'system') return val;
  } catch {
    // localStorage unavailable
  }
  return 'dark';
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(readStored);
  const [resolved, setResolved] = useState<'dark' | 'light'>(() => resolve(readStored()));

  // Apply class to <html> and persist
  useEffect(() => {
    const r = resolve(mode);
    setResolved(r);

    const el = document.documentElement;
    if (r === 'light') {
      el.classList.add('light');
    } else {
      el.classList.remove('light');
    }

    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch {
      // ignore
    }
  }, [mode]);

  // Listen for system preference changes when in system mode
  useEffect(() => {
    if (mode !== 'system') return;

    const mql = window.matchMedia('(prefers-color-scheme: light)');
    const handler = () => {
      const r = resolve('system');
      setResolved(r);
      if (r === 'light') {
        document.documentElement.classList.add('light');
      } else {
        document.documentElement.classList.remove('light');
      }
    };

    mql.addEventListener('change', handler);
    return () => mql.removeEventListener('change', handler);
  }, [mode]);

  const toggle = useCallback(() => {
    setMode((prev) => {
      const idx = CYCLE.indexOf(prev);
      const next = CYCLE[(idx + 1) % CYCLE.length];
      return next ?? 'dark';
    });
  }, []);

  return (
    <ThemeContext.Provider value={{ mode, resolved, toggle }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}
