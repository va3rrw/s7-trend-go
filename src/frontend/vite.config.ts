import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import UnoCSS from 'unocss/vite'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'
import { defineConfig } from 'vitest/config'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
let appVersion = 'dev'
try {
  appVersion = fs.readFileSync(path.resolve(__dirname, '../../VERSION'), 'utf8').trim() || 'dev'
} catch {
  // Keep 'dev' fallback if VERSION file is absent
}

export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  plugins: [
    vue(),
    UnoCSS(),
    ...(!process.env.VITEST ? [wails('./bindings')] : []),
  ],
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/timerGuard.ts', './test/setup.ts'],
  },
})
