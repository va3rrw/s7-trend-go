<template>
  <AppDialog :open="open" :title="$t('dialog.y_axis_settings')" width="560px" @close="emit('close')" @submit="save">
    <table class="datagrid">
      <thead>
        <tr>
          <th>Name</th><th>{{ $t("dialog.minimum") }}</th><th>{{ $t("dialog.maximum") }}</th><th>Expand on overflow</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(axis, i) in axes" :key="i">
          <td><input v-model="axis.name" type="text" /></td>
          <td><input v-model.number="axis.minimum" type="number" /></td>
          <td><input v-model.number="axis.maximum" type="number" /></td>
          <td><input v-model="axis.autoScale" type="checkbox" /></td>
        </tr>
      </tbody>
    </table>

    <template #footer>
      <div style="flex:1" />
      <button class="btn btn-primary" @click="save">{{ $t("buttons.ok") }}</button>
      <button class="btn btn-outline" @click="emit('close')">{{ $t("buttons.cancel") }}</button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { YAxisSettings } from '../types'
import { state, showMessage } from '../store'
import { useI18n } from 'vue-i18n'
import AppDialog from './AppDialog.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; save: [] }>()
const { t } = useI18n()

const axes = ref<YAxisSettings[]>([])

watch(() => props.open, (val) => {
  if (val) {
    axes.value = state.settings.yAxes.map(a => ({ ...a }))
  }
})

function save() {
  for (const axis of axes.value) {
    if (!axis.name.trim()) {
      showMessage(t('dialog.y_axis_settings'), t('prompt.axis_name_required'))
      return
    }
    if (axis.maximum <= axis.minimum) {
      showMessage(t('dialog.y_axis_settings'), t('prompt.invalid_axis_range'))
      return
    }
  }
  state.settings.yAxes = axes.value.map(a => ({ ...a }))
  emit('save')
  emit('close')
}
</script>
