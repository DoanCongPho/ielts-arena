import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],

  build: {
    // The Go API owns the /assets/ URL namespace (test media: Task 1 chart
    // images, diagrams, audio). Vite's default output dir is also "assets",
    // which would collide behind the nginx proxy, so the SPA's own hashed
    // bundles go to /static/ instead.
    assetsDir: 'static',
  },

  server: {
    // Dev mirrors production: the app always calls the API on its own
    // origin and the proxy decides where that goes. Nothing in the app
    // code knows the API's host, in dev or in prod.
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/assets': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
