import UnoCSS from 'unocss/vite'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
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
    setupFiles: ['./test/setup.ts'],
  },
})
