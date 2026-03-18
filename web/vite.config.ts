import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 9374,
    proxy: {
      '/api': 'http://localhost:9375',
      '/ws': { target: 'ws://localhost:9375', ws: true },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
});
