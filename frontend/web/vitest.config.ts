import react from '@vitejs/plugin-react'
import path from 'node:path'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@aether/design-system': path.resolve(__dirname, '../aether_ds/src'),
      '@aether/design-system/styles.css': path.resolve(__dirname, '../aether_ds/src/styles.css'),
      react: path.resolve(__dirname, 'node_modules/react'),
      'react-dom': path.resolve(__dirname, 'node_modules/react-dom'),
    },
    dedupe: ['react', 'react-dom'],
  },
    test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: ['./src/test/setup.ts'],
      server: {
        deps: {
          inline: ['@aether/design-system', '@base-ui/react', '@base-ui/utils'],
        },
      },
    },
  })
