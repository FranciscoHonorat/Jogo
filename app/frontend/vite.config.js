import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// O build vai direto para backend/dist, que o Go embute no binário.
// Em desenvolvimento (npm run dev), a API é redirecionada para o Go na porta 3000.
export default defineConfig({
  plugins: [vue()],
  build: { outDir: '../backend/dist', emptyOutDir: true },
  server: {
    proxy: {
      '/api': 'http://localhost:3000',
      '/events': 'http://localhost:3000',
    },
  },
});
