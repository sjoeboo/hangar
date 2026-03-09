import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import path from 'path'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // Cache static assets (JS, CSS, fonts, images)
        globPatterns: ['**/*.{js,css,html,png,svg,woff2}'],
        // Don't cache API or WebSocket requests
        navigateFallback: '/ui/index.html',
        navigateFallbackAllowlist: [/^\/ui/],
        runtimeCaching: [
          {
            urlPattern: /^\/api\//,
            handler: 'NetworkOnly',
          },
        ],
      },
      manifest: {
        name: 'Hangar',
        short_name: 'Hangar',
        description: 'Terminal session manager for AI coding agents',
        theme_color: '#101825',
        background_color: '#101825',
        display: 'standalone',
        scope: '/ui/',
        start_url: '/ui/',
        icons: [
          {
            src: 'icon-192.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: 'icon-512.png',
            sizes: '512x512',
            type: 'image/png',
          },
          {
            src: 'icon-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },
    }),
  ],
  base: '/ui/',
  build: {
    outDir: '../internal/webui/assets/dist',
    emptyOutDir: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:47437',
      '/hooks': 'http://localhost:47437',
    },
  },
})
