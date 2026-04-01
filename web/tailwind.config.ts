import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bc: {
          bg: 'var(--bc-bg)',
          surface: 'var(--bc-surface)',
          'surface-hover': 'var(--bc-surface-hover)',
          border: 'var(--bc-border)',
          text: 'var(--bc-text)',
          muted: 'var(--bc-muted)',
          accent: 'var(--bc-accent)',
          'accent-hover': 'var(--bc-accent-hover)',
          success: 'var(--bc-success)',
          warning: 'var(--bc-warning)',
          error: 'var(--bc-error)',
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
