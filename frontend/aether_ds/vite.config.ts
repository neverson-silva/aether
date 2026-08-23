import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'

const root = fileURLToPath(new URL('.', import.meta.url))
const fontFiles = [
  ['inter-latin-400-normal.woff2', '@fontsource/inter/files/inter-latin-400-normal.woff2'],
  ['inter-latin-600-normal.woff2', '@fontsource/inter/files/inter-latin-600-normal.woff2'],
  ['inter-latin-700-normal.woff2', '@fontsource/inter/files/inter-latin-700-normal.woff2'],
  ['jetbrains-mono-latin-400-normal.woff2', '@fontsource/jetbrains-mono/files/jetbrains-mono-latin-400-normal.woff2'],
  ['jetbrains-mono-latin-500-normal.woff2', '@fontsource/jetbrains-mono/files/jetbrains-mono-latin-500-normal.woff2'],
] as const

function externalFonts(): Plugin {
  return {
    name: 'external-fonts',
    generateBundle(_options, bundle) {
      const encodedFonts = new Map<string, string>()
      for (const [name, source] of fontFiles) {
        const data = readFileSync(resolve(root, 'node_modules', source))
        this.emitFile({ type: 'asset', fileName: `assets/fonts/${name}`, source: data })
        encodedFonts.set(name, `data:font/woff2;base64,${data.toString('base64')}`)
      }
      for (const output of Object.values(bundle)) {
        if (output.type !== 'asset' || !output.fileName.endsWith('.css') || typeof output.source !== 'string') continue
        for (const [name, encoded] of encodedFonts) {
          output.source = output.source.replaceAll(`url(${encoded})`, `url('./fonts/${name}')`)
        }
      }
    },
  }
}

export default defineConfig({
  plugins: [react(), tailwindcss(), externalFonts()],
  build: {
    cssCodeSplit: true,
    assetsInlineLimit: 0,
    lib: {
      entry: 'src/index.ts',
      formats: ['es'],
      fileName: 'index',
    },
    rollupOptions: {
      external: [
        'react',
        'react-dom',
        'react/jsx-runtime',
        '@base-ui/react',
        '@phosphor-icons/react',
        'tailwind-variants',
      ],
      output: {
        assetFileNames: (asset) => asset.name?.endsWith('.css') ? 'styles.css' : 'assets/[name]-[hash][extname]',
      },
    },
  },
})
