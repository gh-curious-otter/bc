/**
 * Terminal / ANSI color mappings for the TUI.
 *
 * Maps each palette color to the closest standard ANSI color name for
 * terminals that don't support 24-bit color, alongside the true hex
 * value for truecolor-capable terminals.
 */

import { palette } from "./colors.js";

export interface TerminalColor {
  /** Hex value for truecolor terminals */
  hex: string;
  /** Closest ANSI 16-color name as fallback */
  ansi: string;
}

export const terminalColors: Record<string, TerminalColor> = {
  obsidian: { hex: palette.obsidian, ansi: "black" },
  umber: { hex: palette.umber, ansi: "black" },
  bark: { hex: palette.bark, ansi: "black" },
  sandstone: { hex: palette.sandstone, ansi: "white" },
  warmWhite: { hex: palette.warmWhite, ansi: "brightWhite" },
  tangerine: { hex: palette.tangerine, ansi: "red" },
  amberGlow: { hex: palette.amberGlow, ansi: "yellow" },
  peach: { hex: palette.peach, ansi: "brightYellow" },
} as const;
