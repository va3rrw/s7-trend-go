<template>
  <AppDialog :open="open" :title="title" width="400px" @close="emit('close', null)" @submit="ok">
    <p style="margin-bottom: 8px;">{{ text }}</p>
    <template v-if="options?.length">
      <select v-model="selectValue" style="width: 100%;" autofocus>
        <option v-for="opt in options" :key="opt" :value="opt">{{ opt }}</option>
      </select>
    </template>
    <template v-else>
      <input
        id="promptInput"
        v-model="inputValue"
        type="text"
        style="width: 100%;"
        autofocus
        @keydown.enter="ok"
      />
    </template>

    <template #footer>
      <div style="flex:1" />
      <button class="btn btn-primary" @click="ok">{{ $t("buttons.ok") }}</button>
      <button class="btn btn-outline" @click="cancel">{{ $t("buttons.cancel") }}</button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import AppDialog from './AppDialog.vue'

const props = defineProps<{
  open: boolean
  title: string
  text: string
  defaultValue: string
  options?: string[]
}>()
const emit = defineEmits<{ close: [result: string | null] }>()

const inputValue = ref('')
const selectValue = ref('')

watch(() => props.open, (val) => {
  if (val) {
    inputValue.value = props.defaultValue
    selectValue.value = props.defaultValue
    nextTick(() => {
      // Auto-focus the input
      const el = document.querySelector('#promptInput') as HTMLInputElement
      el?.focus()
      el?.select()
    })
  }
})

function ok() {
  const result = props.options?.length ? selectValue.value : inputValue.value
  emit('close', result)
}

function cancel() {
  emit('close', null)
}
</script>
