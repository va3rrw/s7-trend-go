<template>
    <AppDialog
        :open="open"
        :title="$t('dialog.plc_tag_config')"
        width="1040px"
        @close="cancel"
        @submit="save">
        <div class="plc-tags-layout">
            <!-- Left Panel: PLC Connections -->
            <div class="panel plc-panel">
                <div class="panel-header">
                    <span class="panel-title">{{ $t('dialog.plc_config') }}</span>
                    <span
                        class="header-badge"
                        :class="{ 'at-limit': links.length >= MAX_PLC_LINKS }">
                        {{ links.length }} / {{ MAX_PLC_LINKS }} PLCs
                    </span>
                </div>
                <div class="panel-body">
                    <table class="datagrid plc-grid">
                        <thead>
                            <tr>
                                <th style="width: 52px; text-align: center;">{{ $t('dialog.status') }}</th>
                                <th style="width: 120px;">{{ $t('dialog.link') }}</th>
                                <th style="width: 125px;">{{ $t('dialog.ip_address') }}</th>
                                <th style="width: 65px; text-align: center;">{{ $t('dialog.rack') }}</th>
                                <th style="width: 78px; text-align: center;">{{ $t('dialog.cpu_slot') }}</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr
                                v-for="(plc, i) in links"
                                :key="i"
                                :class="{ selected: i === selectedPlcIndex }"
                                @click="selectPlc(i)">
                                <td style="text-align: center;">
                                    <div
                                        :class="[
                                            'status-dot',
                                            { connected: plc.isConnected },
                                        ]"
                                        style="margin: 0 auto;" />
                                </td>
                                <td>
                                    <input
                                        v-model="plc.name"
                                        type="text"
                                        class="table-input"
                                        @focus="onPlcFocus(plc.name)"
                                        @change="onPlcNameChange(i, plc.name)" />
                                </td>
                                <td>
                                    <input
                                        v-model="plc.ipAddress"
                                        type="text"
                                        class="table-input" />
                                </td>
                                <td>
                                    <input
                                        v-model.number="plc.rack"
                                        type="number"
                                        min="0"
                                        max="9"
                                        class="table-input num-input" />
                                </td>
                                <td>
                                    <input
                                        v-model.number="plc.slot"
                                        type="number"
                                        min="0"
                                        max="9"
                                        class="table-input num-input" />
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="panel-footer">
                    <button
                        class="btn btn-outline"
                        :disabled="links.length >= MAX_PLC_LINKS"
                        :title="links.length >= MAX_PLC_LINKS ? $t('prompt.max_plcs_exceeded', [MAX_PLC_LINKS]) : ''"
                        @click="addPlc">
                        {{ $t('buttons.add') }}
                    </button>
                    <button
                        class="btn btn-outline"
                        :disabled="links.length <= MIN_PLC_LINKS || selectedPlcIndex < 0"
                        :title="links.length <= MIN_PLC_LINKS ? $t('prompt.min_plcs_required', [MIN_PLC_LINKS]) : ''"
                        @click="removePlc">
                        {{ $t('buttons.remove') }}
                    </button>
                    <button
                        class="btn btn-outline"
                        :disabled="!activePlc"
                        @click="testConnection">
                        {{ $t('buttons.test_connection') }} ({{ activePlc?.name }})
                    </button>
                </div>
            </div>

            <!-- Right Panel: Tags for Selected PLC -->
            <div class="panel tags-panel">
                <div class="panel-header">
                    <span class="panel-title">
                        {{ activePlc ? $t('dialog.tags_for_plc', [activePlc.name]) : $t('dialog.tags_config') }}
                    </span>
                    <span
                        v-if="activePlc"
                        class="header-badge"
                        :class="{ 'at-limit': activePlcTags.length >= MAX_TAGS_PER_PLC_LINK }">
                        {{ activePlcTags.length }} / {{ MAX_TAGS_PER_PLC_LINK }} {{ $t('menu.tags') }}
                    </span>
                </div>
                <div class="panel-body">
                    <table class="datagrid tags-grid">
                        <thead>
                            <tr>
                                <th style="width: 54px; text-align: center;">{{ $t('grid.on') }}</th>
                                <th style="width: 140px;">{{ $t('dialog.name') }}</th>
                                <th style="width: 120px;">{{ $t('dialog.address') }}</th>
                                <th style="width: 80px;">{{ $t('dialog.type') }}</th>
                                <th style="width: 100px;">{{ $t('dialog.y_axis') }}</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr
                                v-for="(tag, i) in activePlcTags"
                                :key="tag.id"
                                :class="{ selected: i === selectedTagIndex }"
                                :style="{ backgroundColor: tag.color || 'transparent' }"
                                @click="selectTag(i)"
                                @dblclick="openEditor(i)">
                                <td style="text-align: center;">
                                    <input
                                        type="checkbox"
                                        :checked="tag.enabled"
                                        @change="
                                            tag.enabled = (
                                                $event.target as HTMLInputElement
                                            ).checked
                                        " />
                                </td>
                                <td>{{ tag.name }}</td>
                                <td>{{ tag.address }}</td>
                                <td>{{ tag.dataType }}</td>
                                <td>{{ tag.dataType === 'Bool' ? '-' : tag.yAxis }}</td>
                            </tr>
                            <tr v-if="activePlcTags.length === 0">
                                <td
                                    colspan="5"
                                    class="empty-state">
                                    {{ $t('dialog.no_tags_for_plc') }}
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="panel-footer">
                    <button
                        class="btn btn-outline"
                        :disabled="!activePlc || activePlcTags.length >= MAX_TAGS_PER_PLC_LINK"
                        :title="activePlcTags.length >= MAX_TAGS_PER_PLC_LINK ? $t('prompt.max_tags_per_plc_exceeded', [activePlc?.name, MAX_TAGS_PER_PLC_LINK]) : ''"
                        @click="addTag">
                        {{ $t('buttons.add') }}
                    </button>
                    <button
                        class="btn btn-outline"
                        :disabled="selectedTagIndex < 0"
                        @click="editTag">
                        {{ $t('buttons.edit') }}
                    </button>
                    <button
                        class="btn btn-outline"
                        :disabled="selectedTagIndex < 0"
                        @click="removeTag">
                        {{ $t('buttons.remove') }}
                    </button>
                    <button
                        class="btn btn-outline"
                        :disabled="selectedTagIndex <= 0"
                        @click="moveTag(-1)">
                        {{ $t('buttons.move_up') }}
                    </button>
                    <button
                        class="btn btn-outline"
                        :disabled="selectedTagIndex < 0 || selectedTagIndex >= activePlcTags.length - 1"
                        @click="moveTag(1)">
                        {{ $t('buttons.move_down') }}
                    </button>
                </div>
            </div>
        </div>

        <template #footer>
            <button
                class="btn btn-outline"
                @click="loadFromFile">
                {{ $t('menu.load_settings') }}
            </button>
            <button
                class="btn btn-outline"
                @click="saveToFile">
                {{ $t('menu.save_settings') }}
            </button>
            <div style="flex: 1" />
            <button
                class="btn btn-primary"
                @click="save">
                {{ $t('buttons.ok') }}
            </button>
            <button
                class="btn btn-outline"
                @click="cancel">
                {{ $t('buttons.cancel') }}
            </button>
        </template>
    </AppDialog>

    <TagEditorDialog
        :open="editorOpen"
        :tag="editorTag"
        :index="globalEditorIndex"
        :existing-tags="tags"
        :hide-plc="true"
        @save="onEditorSave"
        @close="onEditorClose" />
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import type { PlcLinkSettings, TagSettings, YAxisSettings } from '../types';
import { PALETTE, newId, MIN_PLC_LINKS, MAX_PLC_LINKS, MAX_TAGS_PER_PLC_LINK } from '../types';
import { state, backend, showMessage } from '../store';
import { useI18n } from 'vue-i18n';
import AppDialog from './AppDialog.vue';
import TagEditorDialog from './TagEditorDialog.vue';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: []; save: [] }>();
const { t } = useI18n();

// Working copies while dialog is open
const links = ref<PlcLinkSettings[]>([]);
const tags = ref<TagSettings[]>([]);
const selectedPlcIndex = ref(0);
const selectedTagIndex = ref(-1);

// Active PLC computed
const activePlc = computed<PlcLinkSettings | undefined>(
    () => links.value[selectedPlcIndex.value],
);

// Tags filtered for active PLC
const activePlcTags = computed<TagSettings[]>(() => {
    if (!activePlc.value) return [];
    return tags.value.filter((t) => t.plcLink === activePlc.value?.name);
});

function getPlcTagCount(plcName: string): number {
    return tags.value.filter((t) => t.plcLink === plcName).length;
}

// Sub-dialog state
const editorOpen = ref(false);
const editorTag = ref<TagSettings | null>(null);
const globalEditorIndex = ref(-1);
const pendingNewId = ref<string | null>(null);

let focusedPlcOldName = '';

watch(
    () => props.open,
    (val) => {
        if (val) {
            links.value = state.settings.plcLinks.map((p) => ({ ...p }));
            tags.value = state.settings.tags.map((t) => ({ ...t }));
            selectedPlcIndex.value = 0;
            selectedTagIndex.value = -1;
            focusedPlcOldName = '';
        }
    },
);

function selectPlc(index: number) {
    selectedPlcIndex.value = index;
    selectedTagIndex.value = -1;
}

function selectTag(index: number) {
    selectedTagIndex.value = index;
}

function onPlcFocus(name: string) {
    focusedPlcOldName = name;
}

function onPlcNameChange(index: number, newName: string) {
    const trimmed = newName.trim();
    if (!trimmed) {
        links.value[index].name = focusedPlcOldName || `PLC${index + 1}`;
        return;
    }
    const oldName = focusedPlcOldName || links.value[index]?.name;
    if (oldName && oldName !== trimmed) {
        // Update any tags using the old PLC name
        for (const tag of tags.value) {
            if (tag.plcLink === oldName) {
                tag.plcLink = trimmed;
            }
        }
        focusedPlcOldName = trimmed;
    }
}

function addPlc() {
    if (links.value.length >= MAX_PLC_LINKS) {
        showMessage(t('dialog.plc_config'), t('prompt.max_plcs_exceeded', [MAX_PLC_LINKS]));
        return;
    }
    let nextNum = 1;
    const existingNames = new Set(links.value.map((p) => p.name));
    while (existingNames.has(`PLC${nextNum}`)) {
        nextNum++;
    }
    const newPlc: PlcLinkSettings = {
        name: `PLC${nextNum}`,
        ipAddress: `192.168.0.${nextNum}`,
        rack: 0,
        slot: 1,
        isConnected: false,
    };
    links.value.push(newPlc);
    selectedPlcIndex.value = links.value.length - 1;
    selectedTagIndex.value = -1;
}

function removePlc() {
    if (links.value.length <= MIN_PLC_LINKS || selectedPlcIndex.value < 0) {
        showMessage(t('dialog.plc_config'), t('prompt.min_plcs_required', [MIN_PLC_LINKS]));
        return;
    }
    const removedPlc = links.value[selectedPlcIndex.value];
    if (!removedPlc) return;

    // Remove tags associated with the deleted PLC
    tags.value = tags.value.filter((t) => t.plcLink !== removedPlc.name);

    links.value.splice(selectedPlcIndex.value, 1);
    if (selectedPlcIndex.value >= links.value.length) {
        selectedPlcIndex.value = links.value.length - 1;
    }
    selectedTagIndex.value = -1;
}

function addTag() {
    if (!activePlc.value) return;
    if (activePlcTags.value.length >= MAX_TAGS_PER_PLC_LINK) {
        showMessage(
            t('dialog.plc_tag_config'),
            t('prompt.max_tags_per_plc_exceeded', [activePlc.value.name, MAX_TAGS_PER_PLC_LINK]),
        );
        return;
    }

    const id = newId();
    const lastTag = activePlcTags.value[activePlcTags.value.length - 1];

    const tag: TagSettings = lastTag
        ? {
              ...lastTag,
              id,
              name: `Tag ${tags.value.length + 1}`,
              plcLink: activePlc.value.name,
              color: PALETTE[tags.value.length % PALETTE.length],
              enabled: true,
          }
        : {
              id,
              name: `Tag ${tags.value.length + 1}`,
              plcLink: activePlc.value.name,
              address: 'DB1.DBD0',
              dataType: 'Real',
              yAxis: state.settings.yAxes[0]?.name ?? 'Y-Axis 1',
              color: PALETTE[tags.value.length % PALETTE.length],
              enabled: true,
          };

    tags.value.push(tag);
    pendingNewId.value = id;
    globalEditorIndex.value = tags.value.length - 1;
    editorTag.value = { ...tag };
    editorOpen.value = true;
}

function editTag() {
    if (selectedTagIndex.value < 0 || selectedTagIndex.value >= activePlcTags.value.length) return;
    openEditor(selectedTagIndex.value);
}

function openEditor(activeTagIdx: number) {
    const targetTag = activePlcTags.value[activeTagIdx];
    if (!targetTag) return;
    const gIndex = tags.value.findIndex((t) => t.id === targetTag.id);
    if (gIndex < 0) return;

    globalEditorIndex.value = gIndex;
    editorTag.value = { ...targetTag };
    editorOpen.value = true;
}

function removeTag() {
    if (selectedTagIndex.value < 0 || selectedTagIndex.value >= activePlcTags.value.length) return;
    const targetTag = activePlcTags.value[selectedTagIndex.value];
    if (!targetTag) return;

    const curIndex = selectedTagIndex.value;
    const gIndex = tags.value.findIndex((t) => t.id === targetTag.id);
    if (gIndex >= 0) {
        tags.value.splice(gIndex, 1);
    }

    const remainingCount = activePlcTags.value.length;
    if (remainingCount === 0) {
        selectedTagIndex.value = -1;
    } else if (curIndex >= remainingCount) {
        selectedTagIndex.value = remainingCount - 1;
    } else {
        selectedTagIndex.value = curIndex;
    }
}

function moveTag(delta: number) {
    if (selectedTagIndex.value < 0) return;
    const currentActive = activePlcTags.value;
    const newActiveIdx = selectedTagIndex.value + delta;
    if (newActiveIdx < 0 || newActiveIdx >= currentActive.length) return;

    const tagA = currentActive[selectedTagIndex.value];
    const tagB = currentActive[newActiveIdx];
    if (!tagA || !tagB) return;

    const idxA = tags.value.findIndex((t) => t.id === tagA.id);
    const idxB = tags.value.findIndex((t) => t.id === tagB.id);
    if (idxA >= 0 && idxB >= 0) {
        const temp = tags.value[idxA];
        tags.value[idxA] = tags.value[idxB];
        tags.value[idxB] = temp;
        selectedTagIndex.value = newActiveIdx;
    }
}

function onEditorSave(savedTag: TagSettings) {
    if (globalEditorIndex.value >= 0 && globalEditorIndex.value < tags.value.length) {
        tags.value[globalEditorIndex.value] = savedTag;
    }
    pendingNewId.value = null;
    editorOpen.value = false;
}

function onEditorClose() {
    if (pendingNewId.value && globalEditorIndex.value >= 0) {
        const tag = tags.value[globalEditorIndex.value];
        if (tag && tag.id === pendingNewId.value) {
            tags.value.splice(globalEditorIndex.value, 1);
        }
        pendingNewId.value = null;
    }
    globalEditorIndex.value = -1;
    editorTag.value = null;
    editorOpen.value = false;
}

async function testConnection() {
    if (!activePlc.value) return;
    const link = activePlc.value;
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

async function loadFromFile() {
    try {
        const loaded = await backend()?.LoadSettingsFile(t('menu.load_settings'));
        if (loaded) {
            if (loaded.plcLinks?.length) {
                links.value = loaded.plcLinks.map((p: PlcLinkSettings) => ({ ...p }));
            }
            tags.value = (loaded.tags ?? []).map((t: TagSettings) => ({ ...t }));
            if (loaded.yAxes?.length) {
                state.settings.yAxes = loaded.yAxes.map((a: YAxisSettings) => ({ ...a }));
            }
            if (loaded.pollIntervalMs) {
                state.settings.pollIntervalMs = loaded.pollIntervalMs;
            }
            if (loaded.timeWindowSeconds) {
                state.settings.timeWindowSeconds = loaded.timeWindowSeconds;
            }
            if (loaded.interpolation) {
                state.settings.interpolation = loaded.interpolation;
            }
            selectedPlcIndex.value = 0;
            selectedTagIndex.value = -1;
            showMessage(t('dialog.plc_tag_config'), t('status.loaded_settings'));
        }
    } catch (err: any) {
        showMessage(t('dialog.plc_tag_config'), t('status.error_loading_settings', [err]));
    }
}

async function saveToFile() {
    try {
        const settingsToSave = {
            ...state.settings,
            plcLinks: links.value.map((p) => ({ ...p })),
            tags: tags.value.map((t) => ({ ...t })),
        };
        await backend()?.SaveSettingsFile(settingsToSave, t('menu.save_settings'));
        showMessage(t('dialog.plc_tag_config'), t('status.saved_settings'));
    } catch (err: any) {
        showMessage(t('dialog.plc_tag_config'), t('status.error_saving_settings', [err]));
    }
}

function save() {
    state.settings.plcLinks = links.value.map((p) => ({ ...p }));
    state.settings.tags = tags.value.map((t) => ({ ...t }));
    emit('save');
    emit('close');
}

function cancel() {
    emit('close');
}
</script>

<style scoped>
.plc-tags-layout {
    display: flex;
    gap: 12px;
    height: 542px;
    min-height: 507px;
}

.panel {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border-color, #CBD5E1);
    border-radius: 6px;
    background: #ffffff;
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.plc-panel {
    flex: 0 0 480px;
}

.tags-panel {
    flex: 1 1 auto;
    min-width: 0;
}

.panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: #f8fafc;
    border-bottom: 1px solid var(--border-color, #CBD5E1);
    font-size: 13px;
}

.panel-title {
    font-weight: 600;
    color: #1e293b;
}

.header-badge {
    font-size: 12px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 10px;
    background: #e0f2fe;
    color: #0369a1;
}

.header-badge.at-limit {
    background: #fee2e2;
    color: #b91c1c;
    font-weight: 600;
}

.panel-body {
    flex: 1 1 auto;
    overflow-y: auto;
    overflow-x: hidden;
    background: #ffffff;
}

.panel-footer {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 10px;
    background: #f8fafc;
    border-top: 1px solid var(--border-color, #CBD5E1);
}

.datagrid {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
    font-size: 13px;
}

.datagrid th {
    background-color: #f1f5f9;
    color: #475569;
    font-weight: 500;
    font-size: 12px;
    padding: 6px 8px;
    border-bottom: 1px solid #cbd5e1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.datagrid td {
    padding: 4px 8px;
    border-bottom: 1px solid #f1f5f9;
    color: #1e293b;
    vertical-align: middle;
    font-size: 13px;
}

.datagrid tbody tr:hover {
    background-color: #f8fafc;
}

.datagrid tbody tr.selected {
    background-color: #e0f2fe !important;
    outline: 1px solid #38bdf8;
    outline-offset: -1px;
}

.table-input {
    width: 100%;
    background: #ffffff;
    border: 1px solid #cbd5e1;
    color: #1e293b;
    font-size: 13px;
    padding: 2px 6px;
    border-radius: 3px;
    box-sizing: border-box;
}

.table-input:hover {
    border-color: #94a3b8;
}

.table-input:focus {
    border-color: var(--accent, #0078d7);
    background: #ffffff;
    outline: none;
    box-shadow: 0 0 0 1px var(--accent, #0078d7);
}

.num-input {
    text-align: center;
}

.color-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    margin-right: 6px;
    vertical-align: middle;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.15);
}

.empty-state {
    text-align: center;
    color: #94a3b8;
    padding: 32px 16px;
    font-style: italic;
}
</style>
