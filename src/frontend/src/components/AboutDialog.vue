<template>
    <AppDialog
        :open="open"
        :title="$t('dialog.about_title')"
        width="400px"
        @close="emit('close')">
        <div style="line-height: 1.6">
            <strong>S7 Trend Go v{{ version }}</strong><br />
            {{ $t('dialog.about_desc1') }}<br />
            {{ $t('dialog.about_desc2') }}<br />
            Ken Wang, 2026<br />
            {{ $t('dialog.about_desc3') }}
        </div>

        <template #footer>
            <div style="flex: 1" />
            <button
                class="btn btn-primary"
                @click="emit('close')">
                OK
            </button>
        </template>
    </AppDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import AppDialog from './AppDialog.vue';
import { APP_VERSION } from '../version';
import { backend } from '../store';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();

const version = ref(APP_VERSION);

watch(
    () => props.open,
    async (val) => {
        if (!val) return;
        try {
            const v = await backend()?.GetVersion?.();
            if (v) version.value = v;
        } catch {
            version.value = APP_VERSION;
        }
    },
);
</script>
