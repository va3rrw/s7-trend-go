<template>
  <AppDialog :open="open" :title="title || $t('dialog.exit_confirm_title')" width="440px" @close="emit('cancel')">
    <p style="word-break: break-word; overflow-wrap: anywhere; line-height: 1.5; padding: 4px 0;">
      {{ message || $t('dialog.exit_confirm_msg') }}
    </p>

    <template #footer>
      <button type="button" class="btn btn-outline btn-cancel" @click="emit('cancel')">
        {{ $t('buttons.cancel') }}
      </button>
      <div style="flex: 1" />
      <button type="button" class="btn btn-outline btn-dont-save" @click="emit('dontSave')">
        {{ $t('buttons.dont_save') }}
      </button>
      <button type="button" class="btn btn-primary btn-save" autofocus @click="emit('save')">
        {{ $t('buttons.save') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
import AppDialog from './AppDialog.vue'

defineProps<{
  open: boolean
  title?: string
  message?: string
}>()

const emit = defineEmits<{
  save: []
  dontSave: []
  cancel: []
}>()
</script>
