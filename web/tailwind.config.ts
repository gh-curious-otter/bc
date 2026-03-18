import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bc: {
          bg: 'var(--bc-bg)',
          surface: 'var(--bc-surface)',
          border: 'var(--bc-border)',
          text: 'var(--bc-text)',
          muted: 'var(--bc-muted)',
          accent: 'var(--bc-accent)',
          success: 'var(--bc-success)',
          warning: 'var(--bc-warning)',
          error: 'var(--bc-error)',
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
