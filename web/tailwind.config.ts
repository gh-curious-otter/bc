import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      keyframes: {
        'slide-in': {
          '0%': { opacity: '0', transform: 'translateX(1rem)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
      },
      animation: {
        'slide-in': 'slide-in 0.2s ease-out',
      },
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
