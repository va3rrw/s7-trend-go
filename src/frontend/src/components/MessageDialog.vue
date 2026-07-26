<template>
  <AppDialog :open="open" :title="title" width="420px" @close="emit('close', false)" @submit="emit('close', true)">
    <p style="word-break: break-word; overflow-wrap: anywhere;">{{ message }}</p>

    <template #footer>
      <div style="flex:1" />
      <button v-if="showCancel" class="btn btn-outline" @click="emit('close', false)">{{ $t("buttons.cancel") }}</button>
      <button class="btn btn-primary" autofocus @click="emit('close', true)">{{ $t("buttons.ok") }}</button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
import AppDialog from './AppDialog.vue'

defineProps<{
  open: boolean
  title: string
  message: string
  showCancel?: boolean
}>()
const emit = defineEmits<{ close: [result: boolean] }>()
</script>
