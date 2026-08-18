<template>
    <AppDialog
        :open="open"
        :title="$t('dialog.tags_config')"
        width="850px"
        @close="cancel"
        @submit="save">
        <table class="datagrid">
            <thead>
                <tr>
                    <th>{{ $t('grid.on') }}</th>
                    <th>{{ $t('dialog.name') }}</th>
                    <th>{{ $t('grid.plc') }}</th>
                    <th>{{ $t('dialog.address') }}</th>
                    <th>{{ $t('dialog.type') }}</th>
                    <th>{{ $t('dialog.y_axis') }}</th>
                </tr>
            </thead>
            <tbody>
                <tr
                    v-for="(tag, i) in tags"
                    :key="tag.id"
                    :class="{ selected: i === selectedIndex }"
                    :style="{ backgroundColor: tag.color || 'transparent' }"
                    @click="selectRow(i)"
                    @dblclick="openEditor(i)">
                    <td>
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
                    <td>{{ tag.plcLink }}</td>
                    <td>{{ tag.address }}</td>
                    <td>{{ tag.dataType }}</td>
                    <td>{{ tag.yAxis }}</td>
                </tr>
            </tbody>
        </table>

        <template #footer>
            <button
                class="btn btn-outline"
                @click="addTag">
                {{ $t('buttons.add') }}
            </button>
            <button
                class="btn btn-outline"
                :disabled="!hasSelectedTag"
                @click="editTag">
                {{ $t('buttons.edit') }}
            </button>
            <button
                class="btn btn-outline"
                :disabled="!hasSelectedTag"
                @click="removeTag">
                {{ $t('buttons.remove') }}
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
        :index="editorIndex"
        @save="onEditorSave"
        @close="onEditorClose" />
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import type { TagSettings } from '../types';
import { PALETTE, newId } from '../types';
import { state } from '../store';
import AppDialog from './AppDialog.vue';
import TagEditorDialog from './TagEditorDialog.vue';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: []; save: [] }>();

// Working copy of tags while the dialog is open
const tags = ref<TagSettings[]>([]);
const selectedIndex = ref(-1);

const hasSelectedTag = computed(
    () => selectedIndex.value >= 0 && selectedIndex.value < tags.value.length,
);

// Sub-dialog state
const editorOpen = ref(false);
const editorTag = ref<TagSettings | null>(null);
const editorIndex = ref(-1);
const pendingNewId = ref<string | null>(null);

watch(
    () => props.open,
    (val) => {
        if (val) {
            tags.value = state.settings.tags.map((t) => ({ ...t }));
            selectedIndex.value = -1;
        }
    },
);

function selectRow(index: number) {
    selectedIndex.value = index;
}

function addTag() {
    const id = newId();
    // Prefer selected row, else last tag, so successive adds keep previous field values
    const template =
        (hasSelectedTag.value ? tags.value[selectedIndex.value] : null) ??
        tags.value[tags.value.length - 1] ??
        null;

    const tag: TagSettings = template
        ? {
              ...template,
              id,
              name: `Tag ${tags.value.length + 1}`,
              color: PALETTE[tags.value.length % PALETTE.length],
              enabled: true,
          }
        : {
              id,
              name: `Tag ${tags.value.length + 1}`,
              plcLink: state.settings.plcLinks[0]?.name ?? 'PLC1',
              address: 'DB1.DBD0',
              dataType: 'Real',
              stringLength: 20,
              yAxis: state.settings.yAxes[0]?.name ?? 'Y-Axis 1',
              color: PALETTE[tags.value.length % PALETTE.length],
              enabled: true,
          };

    tags.value.push(tag);
    selectedIndex.value = tags.value.length - 1;
    pendingNewId.value = id;
    openEditor(selectedIndex.value);
}

function editTag() {
    if (hasSelectedTag.value) openEditor(selectedIndex.value);
}

function removeTag() {
    if (!hasSelectedTag.value) return;
    tags.value.splice(selectedIndex.value, 1);
    selectedIndex.value = -1;
}

function openEditor(index: number) {
    if (index < 0 || index >= tags.value.length) return;
    editorIndex.value = index;
    editorTag.value = { ...tags.value[index] };
    editorOpen.value = true;
}

function onEditorSave(tag: TagSettings) {
    if (editorIndex.value >= 0 && editorIndex.value < tags.value.length) {
        tags.value[editorIndex.value] = tag;
    }
    pendingNewId.value = null;
    editorOpen.value = false;
}

function onEditorClose() {
    // If cancelling a newly-added tag, remove it
    if (pendingNewId.value && editorIndex.value >= 0) {
        const tag = tags.value[editorIndex.value];
        if (tag && tag.id === pendingNewId.value) {
            tags.value.splice(editorIndex.value, 1);
        }
        pendingNewId.value = null;
    }
    selectedIndex.value = -1;
    editorIndex.value = -1;
    editorTag.value = null;
    editorOpen.value = false;
}

function save() {
    state.settings.tags = tags.value.map((t) => ({ ...t }));
    emit('save');
    emit('close');
}

function cancel() {
    emit('close');
}
</script>
