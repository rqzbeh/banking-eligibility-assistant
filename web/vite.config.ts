import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.svg'],
      manifest: {
        name: 'دستیار اهلیت بانکی',
        short_name: 'اهلیت بانکی',
        description: 'بررسی اهلیت، افر شخصی‌سازی‌شده و تحلیل شکاف برای کارمندان شعبه',
        lang: 'fa',
        dir: 'rtl',
        theme_color: '#0b6b5c',
        background_color: '#f3f0e8',
        display: 'standalone',
        start_url: '/',
        icons: [
          {
            src: '/favicon.svg',
            sizes: 'any',
            type: 'image/svg+xml',
            purpose: 'any maskable',
          },
        ],
      },
      workbox: {
        navigateFallback: '/index.html',
        runtimeCaching: [
          {
            urlPattern: ({ url }) => url.pathname.startsWith('/api/'),
            handler: 'NetworkOnly',
          },
        ],
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      '/api/health': 'http://localhost:8080',
      '/api/identity': 'http://localhost:8080',
      '/api/financial': 'http://localhost:8080',
      '/api/rbci': 'http://localhost:8080',
      '/api/products': 'http://localhost:8080',
      '/api/circulars': 'http://localhost:8080',
      '/api/match': 'http://localhost:8080',
      '/api/agent': 'http://localhost:8501',
    },
  },
})
