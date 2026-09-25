import path from 'node:path'
import process from 'node:process'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')
  return {
    plugins: [tanstackRouter({ autoCodeSplitting: true }), react(), tailwindcss()],
    server: {
      port: 5174,
      allowedHosts: true,
      proxy: {
        '/api/': { target: env.VITE_API_TARGET || 'http://127.0.0.1:8080', ws: true },
      },
      fs: {
        allow: [
          path.resolve(import.meta.dirname),
          path.resolve(import.meta.dirname, '../elisyum_ds'),
        ],
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(import.meta.dirname, './src'),
        '@aether/elisyum-ds': path.resolve(import.meta.dirname, '../elisyum_ds/src'),
        react: path.resolve(import.meta.dirname, 'node_modules/react'),
        'react-dom': path.resolve(import.meta.dirname, 'node_modules/react-dom'),
      },
      dedupe: ['react', 'react-dom'],
    },
  }
})
