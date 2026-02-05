import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0', // Listen on all interfaces for remote access
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
        // Configure proxy for SSE (Server-Sent Events)
        configure: (proxy, _options) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            // Ensure proper headers for SSE
            if (req.url?.includes('/events')) {
              proxyReq.setHeader('Accept', 'text/event-stream');
              proxyReq.setHeader('Cache-Control', 'no-cache');
            }
          });
          proxy.on('proxyRes', (proxyRes, req) => {
            // Ensure SSE headers are preserved and buffering is disabled
            if (req.url?.includes('/events')) {
              proxyRes.headers['x-accel-buffering'] = 'no';
              proxyRes.headers['cache-control'] = 'no-cache';
            }
          });
        },
      }
    }
  }
})
