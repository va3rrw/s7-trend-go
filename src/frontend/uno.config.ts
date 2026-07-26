import { defineConfig, presetUno, presetAttributify, presetIcons } from 'unocss'
import ixIcons from '@iconify/json/json/ix.json'

export default defineConfig({
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      collections: {
        ix: ixIcons,
      },
    }),
  ],
})
