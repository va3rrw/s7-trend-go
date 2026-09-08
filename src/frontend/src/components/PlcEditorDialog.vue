<template>
    <AppDialog
        :open="open"
        :title="$t('dialog.plc_properties')"
        width="440px"
        @close="emit('close')"
        @submit="save">
        <div class="form-grid">
            <div class="form-row">
                <label>{{ $t('dialog.link') }}</label>
                <input v-model="form.name" type="text" />
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.ip_address') }}</label>
                <input v-model="form.ipAddress" type="text" />
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.rack') }}</label>
                <input
                    v-model.number="form.rack"
                    type="number"
                    min="0"
                    max="9" />
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.cpu_slot') }}</label>
                <input
                    v-model.number="form.slot"
                    type="number"
                    min="0"
                    max="9" />
            </div>
        </div>

        <template #footer>
            <button
                class="btn btn-outline"
                :disabled="!form.name.trim()"
                @click="testConnection">
                {{ $t('buttons.test_connection') }}
            </button>
            <div style="flex: 1" />
            <button class="btn btn-primary" @click="save">
                {{ $t('buttons.ok') }}
            </button>
            <button class="btn btn-outline" @click="emit('close')">
                {{ $t('buttons.cancel') }}
            </button>
        </template>
    </AppDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { PlcLinkSettings } from '../types';
import { backend, showMessage, state } from '../store';
import AppDialog from './AppDialog.vue';

const props = defineProps<{
    open: boolean;
    plc: PlcLinkSettings | null;
    existingNames: string[];
    originalName: string;
}>();

const emit = defineEmits<{
    close: [];
    save: [plc: PlcLinkSettings];
}>();

const { t } = useI18n();

const form = ref<PlcLinkSettings>({
    name: '',
    ipAddress: '',
    rack: 0,
    slot: 1,
    isConnected: false,
});

watch(
    () => props.open,
    (open) => {
        if (open && props.plc) {
            form.value = { ...props.plc };
        }
    },
    { immediate: true },
);

function save() {
    const plc = { ...form.value };
    plc.name = plc.name.trim();
    plc.ipAddress = plc.ipAddress.trim();

    if (!plc.name) {
        showMessage(t('dialog.plc_properties'), t('prompt.plc_name_required'));
        return;
    }
    if (!plc.ipAddress) {
        showMessage(t('dialog.plc_properties'), t('prompt.ip_required'));
        return;
    }
    if (
        !Number.isInteger(plc.rack) ||
        plc.rack < 0 ||
        plc.rack > 9 ||
        !Number.isInteger(plc.slot) ||
        plc.slot < 0 ||
        plc.slot > 9
    ) {
        showMessage(t('dialog.plc_properties'), t('prompt.invalid_plc_position'));
        return;
    }

    const duplicate = props.existingNames.some(
        (name) =>
            name.toLowerCase() !== props.originalName.toLowerCase() &&
            name.toLowerCase() === plc.name.toLowerCase(),
    );
    if (duplicate) {
        showMessage(t('dialog.plc_properties'), t('prompt.duplicate_plc_name'));
        return;
    }

    emit('save', plc);
}

async function testConnection() {
    const plc = { ...form.value, name: form.value.name.trim(), ipAddress: form.value.ipAddress.trim() };
    if (!plc.name || !plc.ipAddress) return;
    try {
        await backend()?.TestConnection(plc);
        form.value.isConnected = true;
        state.statusMessage = t('status.connected_to', [plc.name, plc.ipAddress]);
        showMessage(t('dialog.plc_properties'), state.statusMessage);
    } catch (err) {
        form.value.isConnected = false;
        state.statusMessage = t('status.test_connection_failed', [err]);
        showMessage(t('dialog.plc_properties'), t('status.connection_failed', [err]));
    }
}
</script>
