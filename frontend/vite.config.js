import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../backend/static/dist',
    emptyOutDir: true
  },
  server: {
    proxy: {
      '/admin': 'http://localhost:8080'
    }
  }
})
