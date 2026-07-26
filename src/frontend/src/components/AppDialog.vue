<template>
  <dialog
    ref="dialogRef"
    class="app-dialog"
    :style="width ? { width: `min(90vw, ${width})` } : undefined"
    @cancel="onCancel"
    @keydown="onKeydown"
  >
    <div class="modal-content" :style="width ? { width } : undefined">
      <div class="modal-header">{{ title }}</div>
      <div class="modal-body">
        <slot />
      </div>
      <div class="modal-footer" v-if="$slots.footer">
        <slot name="footer" />
      </div>
    </div>
  </dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  open: boolean
  title: string
  width?: string
}>()
const emit = defineEmits<{ close: []; submit: [] }>()

const dialogRef = ref<HTMLDialogElement>()

watch(() => props.open, (val) => {
  const el = dialogRef.value
  if (!el) return
  if (val && !el.open) {
    try { el.showModal() } catch { el.setAttribute('open', '') }
  } else if (!val && el.open) {
    try { el.close() } catch { el.removeAttribute('open') }
  }
})

function onCancel(e: Event) {
  e.preventDefault()
  emit('close')
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    const tag = (e.target as HTMLElement)?.tagName
    // Let buttons handle their own Enter key (to click) and textareas handle newlines
    if (tag !== 'TEXTAREA' && tag !== 'BUTTON') {
      e.preventDefault()
      emit('submit')
    }
  }
}
</script>
