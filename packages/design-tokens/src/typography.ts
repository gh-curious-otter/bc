/**
 * Typography tokens — font families, sizes, and weights.
 */

export const fontFamily = {
  /** Body text */
  body: "'Inter', system-ui, -apple-system, sans-serif",
  /** Headings */
  heading: "'Space Grotesk', system-ui, -apple-system, sans-serif",
  /** Code / monospace */
  code: "'Space Mono', ui-monospace, 'Cascadia Code', 'Fira Code', monospace",
} as const;

/** Font sizes in rem, keyed by t-shirt size. */
export const fontSize = {
  xs: "0.75rem",
  sm: "0.875rem",
  base: "1rem",
  lg: "1.125rem",
  xl: "1.25rem",
  "2xl": "1.5rem",
  "3xl": "1.875rem",
  "4xl": "2.25rem",
} as const;

export const fontWeight = {
  regular: 400,
  medium: 500,
  semibold: 600,
  bold: 700,
} as const;
