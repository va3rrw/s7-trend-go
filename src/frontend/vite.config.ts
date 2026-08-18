import UnoCSS from 'unocss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [
    vue(),
    UnoCSS(),
  ],
  server: {
    host: '0.0.0.0',
    port: 34115,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/setup.ts'],
  },
})

