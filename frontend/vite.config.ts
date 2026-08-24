import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The Go API listens on :8080 by default. During development the Vite dev
// server (http://localhost:5173) proxies any request beginning with /api to
// the backend so the browser makes same-origin requests.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
