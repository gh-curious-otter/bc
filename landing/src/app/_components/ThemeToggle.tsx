"use client";

import { Moon, Sun } from "lucide-react";
import { useTheme } from "../_contexts/ThemeContext";

export function ThemeToggle() {
  const { resolvedTheme, toggleTheme } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      aria-label={`Switch to ${resolvedTheme === "light" ? "dark" : "light"} mode`}
      className="inline-flex h-10 w-10 items-center justify-center rounded-full hover:bg-accent transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary"
      title={`Current theme: ${resolvedTheme}`}
    >
      {resolvedTheme === "light" ? (
        <Sun size={20} className="text-foreground" aria-hidden="true" />
      ) : (
        <Moon size={20} className="text-foreground" aria-hidden="true" />
      )}
    </button>
  );
}
