import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

export type ThemeMode = "solar-flare" | "dark" | "light";

interface ThemeContextValue {
  mode: ThemeMode;
  setTheme: (mode: ThemeMode) => void;
  toggle: () => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  mode: "solar-flare",
  setTheme: () => {},
  toggle: () => {},
});

const STORAGE_KEY = "bc-theme";
const CYCLE: ThemeMode[] = ["solar-flare", "dark", "light"];

const LABELS: Record<ThemeMode, string> = {
  "solar-flare": "Solar Flare",
  dark: "Dark",
  light: "Light",
};

function readStored(): ThemeMode {
  try {
    const val = localStorage.getItem(STORAGE_KEY);
    if (val === "solar-flare" || val === "dark" || val === "light") return val;
    // Migrate old "system" preference
    if (val === "system") return "solar-flare";
  } catch {
    // localStorage unavailable
  }
  return "solar-flare";
}

function applyTheme(mode: ThemeMode) {
  const el = document.documentElement;
  // Remove all theme classes
  el.classList.remove("dark", "light");
  // Apply the right class (solar-flare uses :root defaults, no class needed)
  if (mode === "dark") {
    el.classList.add("dark");
  } else if (mode === "light") {
    el.classList.add("light");
  }
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(readStored);

  useEffect(() => {
    applyTheme(mode);
    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch {
      // ignore
    }
  }, [mode]);

  const setTheme = useCallback((m: ThemeMode) => {
    setMode(m);
  }, []);

  const toggle = useCallback(() => {
    setMode((prev) => {
      const idx = CYCLE.indexOf(prev);
      const next = CYCLE[(idx + 1) % CYCLE.length];
      return next ?? "solar-flare";
    });
  }, []);

  return (
    <ThemeContext.Provider value={{ mode, setTheme, toggle }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}

export { LABELS as THEME_LABELS };
