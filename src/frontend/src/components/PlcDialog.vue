<template>
    <AppDialog
        :open="open"
        :title="$t('dialog.plc_config')"
        width="640px"
        @close="emit('close')"
        @submit="save">
        <table class="datagrid">
            <thead>
                <tr>
                    <th>{{ $t('dialog.status') }}</th>
                    <th>{{ $t('dialog.link') }}</th>
                    <th>{{ $t('dialog.ip_address') }}</th>
                    <th>{{ $t('dialog.rack') }}</th>
                    <th>{{ $t('dialog.cpu_slot') }}</th>
                </tr>
            </thead>
            <tbody>
                <tr
                    v-for="(plc, i) in links"
                    :key="i"
                    :class="{ selected: i === selectedIndex }"
                    @click="selectRow(i)">
                    <td>
                        <div
                            :class="[
                                'status-dot',
                                { connected: plc.isConnected },
                            ]" />
                    </td>
                    <td>
                        <input
                            v-model="plc.name"
                            type="text" />
                    </td>
                    <td>
                        <input
                            v-model="plc.ipAddress"
                            type="text" />
                    </td>
                    <td>
                        <input
                            v-model.number="plc.rack"
                            type="number"
                            min="0"
                            max="9"
                            style="width: 52px" />
                    </td>
                    <td>
                        <input
                            v-model.number="plc.slot"
                            type="number"
                            min="0"
                            max="9"
                            style="width: 52px" />
                    </td>
                </tr>
            </tbody>
        </table>

        <template #footer>
            <button
                class="btn btn-outline"
                @click="testConnection">{{ $t("buttons.test_connection") }}</button>
            <div style="flex: 1" />
            <button
                class="btn btn-primary"
                @click="save">
                {{ $t('buttons.ok') }}
            </button>
            <button
                class="btn btn-outline"
                @click="emit('close')">
                {{ $t('buttons.cancel') }}
            </button>
        </template>
    </AppDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import type { PlcLinkSettings } from '../types';
import { state, backend, showMessage } from '../store';
import { useI18n } from 'vue-i18n';
import AppDialog from './AppDialog.vue';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: []; save: [] }>();

// Local editable copy
const links = ref<PlcLinkSettings[]>([]);
const selectedIndex = ref(0);
const { t } = useI18n();

watch(
    () => props.open,
    (val) => {
        if (val) {
            links.value = state.settings.plcLinks.map((p) => ({ ...p }));
            selectedIndex.value = 0;
        }
    },
);

function selectRow(index: number) {
    selectedIndex.value = index;
}

function save() {
    state.settings.plcLinks = links.value.map((p) => ({ ...p }));
    emit('save');
    emit('close');
}

async function testConnection() {
    const link = links.value[selectedIndex.value];
    if (!link) return;
    try {
        await backend()?.TestConnection(link);
        link.isConnected = true;
        state.statusMessage = t('status.connected_to', [link.name, link.ipAddress]);
        showMessage(t('dialog.plc_config'), state.statusMessage);
    } catch (err: any) {
        link.isConnected = false;
        state.statusMessage = t('status.test_connection_failed', [err]);
        showMessage(t('dialog.plc_config'), t('status.connection_failed', [err]));
    }
}
</script>
