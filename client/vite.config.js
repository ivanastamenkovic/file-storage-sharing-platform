import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const gateway = process.env.GATEWAY_URL || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: gateway,
        changeOrigin: true,
      },
    },
  },
})