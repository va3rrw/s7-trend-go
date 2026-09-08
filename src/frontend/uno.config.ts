import { defineConfig, presetUno, presetAttributify, presetIcons } from 'unocss'
import ixIcons from '@iconify/json/json/ix.json'
import mdiIcons from '@iconify/json/json/mdi.json'

export default defineConfig({
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      collections: {
        ix: ixIcons,
        mdi: mdiIcons,
      },
    }),
  ],
})
