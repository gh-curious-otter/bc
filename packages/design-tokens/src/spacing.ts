/**
 * Spacing scale in pixels (base-4 progression).
 *
 * Usage:  `spacing[4]` -> "16px"
 */

const values = [4, 8, 12, 16, 24, 32, 48, 64] as const;

export type SpacingValue = (typeof values)[number];

/** Map from numeric step to pixel string, e.g. `spacing[16] === "16px"`. */
export const spacing: Record<SpacingValue, string> = {
  4: "4px",
  8: "8px",
  12: "12px",
  16: "16px",
  24: "24px",
  32: "32px",
  48: "48px",
  64: "64px",
} as const;

/** Raw numeric values for programmatic use. */
export const spacingValues = values;
