/**
 * Semantic color mappings built on top of the Solar Flare palette.
 *
 * Use these tokens instead of raw palette values so that themes can be
 * swapped without touching component code.
 */

import { palette } from "./colors.js";

export interface SemanticTheme {
  /* Backgrounds */
  bgBase: string;
  bgSurface: string;
  bgElevated: string;

  /* Foregrounds / text */
  fgDefault: string;
  fgMuted: string;

  /* Accent */
  accentPrimary: string;
  accentHover: string;
  accentSubtle: string;

  /* Borders */
  border: string;
}

export const darkTheme: SemanticTheme = {
  bgBase: palette.obsidian,
  bgSurface: palette.umber,
  bgElevated: palette.bark,

  fgDefault: palette.warmWhite,
  fgMuted: palette.sandstone,

  accentPrimary: palette.tangerine,
  accentHover: palette.amberGlow,
  accentSubtle: palette.peach,

  border: palette.sandstone,
};

export const lightTheme: SemanticTheme = {
  bgBase: palette.warmWhite,
  bgSurface: "#EDE8E3",
  bgElevated: "#FFFFFF",

  fgDefault: palette.obsidian,
  fgMuted: palette.sandstone,

  accentPrimary: palette.tangerine,
  accentHover: palette.amberGlow,
  accentSubtle: palette.peach,

  border: palette.sandstone,
};
