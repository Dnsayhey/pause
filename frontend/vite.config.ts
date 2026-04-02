import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import pkg from './package.json';

export default defineConfig({
  plugins: [tailwindcss()],
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(pkg.version)
  },
  build: {
    outDir: 'dist'
  }
});
