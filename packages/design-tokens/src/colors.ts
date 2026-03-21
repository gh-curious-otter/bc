/**
 * Solar Flare color palette.
 *
 * This is the single source of truth for all raw color values used
 * across the bc UI (web, TUI, docs). Consumers should prefer the
 * semantic mappings in `./semantic.ts` over reaching for these
 * primitives directly.
 */

export const palette = {
  /** Near-black base — deepest background tone */
  obsidian: "#0C0A08",
  /** Dark brown — elevated surface backgrounds */
  umber: "#1E1A16",
  /** Warm dark brown — card / panel backgrounds */
  bark: "#2A2420",
  /** Muted warm grey — secondary text, borders */
  sandstone: "#8C7E72",
  /** Off-white with warm undertone — primary text on dark */
  warmWhite: "#F5F0EB",

  /** Vivid orange — primary accent, CTAs */
  tangerine: "#EA580C",
  /** Lighter orange — hover states, highlights */
  amberGlow: "#FB923C",
  /** Soft peach — subtle accent, tags */
  peach: "#FDBA74",
} as const;

export type PaletteColor = keyof typeof palette;
