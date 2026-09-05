import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from "path"

const backendTarget = process.env.VITE_BACKEND_PROXY_TARGET || 'http://localhost:7860'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5174,
    proxy: {
      '/api':       { target: backendTarget, changeOrigin: true, timeout: 0 },
      '/v1':        { target: backendTarget, changeOrigin: true, timeout: 0 },
      '/anthropic': { target: backendTarget, changeOrigin: true, timeout: 0 },
      '/v1beta':    { target: backendTarget, changeOrigin: true, timeout: 0 },
    }
  }
})
