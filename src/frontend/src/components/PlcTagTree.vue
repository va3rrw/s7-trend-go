<template>
    <aside
        class="plc-tree-panel"
        :style="{ width: `${panelWidth}px`, flexBasis: `${panelWidth}px` }">
        <div class="tree-panel-header">
            <span class="tree-panel-title">{{ $t('dialog.plc_tag_tree') }}</span>
            <div class="tree-header-actions">
                <button
                    class="tree-icon-button"
                    type="button"
                    :disabled="plcLinks.length >= MAX_PLC_LINKS"
                    :title="$t('buttons.add_plc')"
                    @click="addPlc">
                    <span class="i-ix-plc-device icon-glyph" aria-hidden="true" />
                </button>
                <button
                    class="tree-icon-button"
                    type="button"
                    :title="$t('buttons.hide_panel')"
                    @click="emit('hide')">
                    <span class="i-ix-chevron-left icon-glyph" aria-hidden="true" />
                </button>
            </div>
        </div>

        <div class="tree-scroll" role="tree">
            <div v-if="!plcLinks.length" class="tree-empty">
                {{ $t('dialog.no_plcs') }}
            </div>
            <ul v-else class="tree-list">
                <li v-for="plc in plcLinks" :key="plc.name" class="tree-branch">
                    <div
                        class="tree-node tree-plc-node"
                        :class="{ selected: selectedPlcName === plc.name }"
                        role="treeitem"
                        :aria-expanded="isExpanded(plc.name)">
                        <button
                            class="tree-toggle"
                            type="button"
                            :aria-label="isExpanded(plc.name) ? $t('buttons.collapse') : $t('buttons.expand')"
                            @click.stop="toggleExpanded(plc.name)">
                            <span
                                class="icon-glyph"
                                :class="isExpanded(plc.name) ? 'i-ix-chevron-down' : 'i-ix-chevron-right'"
                                aria-hidden="true" />
                        </button>
                        <span
                            class="status-dot tree-status-dot"
                            :class="{ connected: plc.isConnected }"
                            :title="plc.isConnected ? $t('dialog.connected') : $t('dialog.disconnected')" />
                        <button
                            class="tree-label tree-plc-label"
                            type="button"
                            @click.stop="selectedPlcName = plc.name"
                            @dblclick.stop="openPlcEditor(plc)">
                            {{ plc.name }}
                        </button>
                        <span class="tree-node-meta">{{ tagCount(plc.name) }}</span>
                        <button
                            class="tree-node-action tree-add-tag-action"
                            type="button"
                            :disabled="tagCount(plc.name) >= MAX_TAGS_PER_PLC_LINK"
                            :title="$t('buttons.add_tag')"
                            @click.stop="addTag(plc)">
                            <span class="i-ix-tag-plus icon-glyph" aria-hidden="true" />
                        </button>
                        <button
                            class="tree-node-action"
                            type="button"
                            :title="$t('buttons.remove')"
                            @click.stop="removePlc(plc)">
                            <span class="i-ix-trashcan icon-glyph" aria-hidden="true" />
                        </button>
                    </div>

                    <ul v-show="isExpanded(plc.name)" class="tree-children" role="group">
                        <li
                            v-for="tag in tagsFor(plc.name)"
                            :key="tag.id"
                            class="tree-node tree-tag-node"
                            :class="{ selected: selectedTagId === tag.id }"
                            role="treeitem"
                            @click.stop="selectedTagId = tag.id; selectedPlcName = tag.plcLink"
                            @dblclick.stop="openTagEditor(tag)">
                            <span class="tree-indent" />
                            <input
                                class="tree-visibility"
                                type="checkbox"
                                :checked="tag.enabled"
                                :title="$t('grid.on')"
                                @click.stop
                                @change="toggleTag(tag, $event)" />
                            <span
                                class="tree-color-dot"
                                :style="{ backgroundColor: tag.color || '#94a3b8' }" />
                            <button
                                class="tree-label tree-tag-label"
                                type="button"
                                @click.stop="selectedTagId = tag.id; selectedPlcName = tag.plcLink"
                                @dblclick.stop="openTagEditor(tag)">
                                {{ tag.name }}
                            </button>
                            <span class="tree-node-meta">{{ tag.dataType }}</span>
                            <button
                                class="tree-node-action"
                                type="button"
                                :title="$t('buttons.remove')"
                                @click.stop="removeTag(tag)">
                                <span class="i-ix-trashcan icon-glyph" aria-hidden="true" />
                            </button>
                        </li>
                        <li v-if="!tagsFor(plc.name).length" class="tree-child-empty">
                            {{ $t('dialog.no_tags_for_plc') }}
                        </li>
                    </ul>
                </li>
            </ul>
        </div>
    </aside>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { PlcLinkSettings, TagSettings } from '../types';
import { MAX_PLC_LINKS, MAX_TAGS_PER_PLC_LINK, MIN_PLC_LINKS, PALETTE, newId } from '../types';
import { showMessage, state } from '../store';

const props = defineProps<{
    plcLinks: PlcLinkSettings[];
    tags: TagSettings[];
    panelWidth: number;
}>();

const emit = defineEmits<{
    hide: [];
    settingsChange: [];
    editPlc: [plc: PlcLinkSettings, isNew: boolean];
    editTag: [tag: TagSettings, isNew: boolean];
}>();

const { t } = useI18n();
const expandedPlcs = ref(new Set<string>());
const selectedPlcName = ref('');
const selectedTagId = ref('');

watch(
    () => props.plcLinks.map((plc) => plc.name),
    (names) => {
        const next = new Set(expandedPlcs.value);
        names.forEach((name) => {
            if (!expandedPlcs.value.has(name)) next.add(name);
        });
        for (const name of next) {
            if (!names.includes(name)) next.delete(name);
        }
        expandedPlcs.value = next;
        if (!names.includes(selectedPlcName.value)) {
            selectedPlcName.value = names[0] ?? '';
        }
    },
    { immediate: true },
);

function tagsFor(plcName: string) {
    return props.tags.filter((tag) => tag.plcLink === plcName);
}

function tagCount(plcName: string) {
    return tagsFor(plcName).length;
}

function isExpanded(plcName: string) {
    return expandedPlcs.value.has(plcName);
}

function toggleExpanded(plcName: string) {
    const next = new Set(expandedPlcs.value);
    if (next.has(plcName)) next.delete(plcName);
    else next.add(plcName);
    expandedPlcs.value = next;
    selectedPlcName.value = plcName;
}

function openPlcEditor(plc: PlcLinkSettings) {
    selectedPlcName.value = plc.name;
    emit('editPlc', plc, false);
}

function openTagEditor(tag: TagSettings) {
    selectedTagId.value = tag.id;
    selectedPlcName.value = tag.plcLink;
    emit('editTag', tag, false);
}

function addPlc() {
    if (props.plcLinks.length >= MAX_PLC_LINKS) {
        showMessage(t('dialog.plc_properties'), t('prompt.max_plcs_exceeded', [MAX_PLC_LINKS]));
        return;
    }
    const names = new Set(props.plcLinks.map((plc) => plc.name));
    let index = 1;
    while (names.has(`PLC${index}`)) index++;
    const plc: PlcLinkSettings = {
        name: `PLC${index}`,
        ipAddress: `192.168.0.${index}`,
        rack: 0,
        slot: 1,
        isConnected: false,
    };
    props.plcLinks.push(plc);
    expandedPlcs.value = new Set([...expandedPlcs.value, plc.name]);
    selectedPlcName.value = plc.name;
    emit('settingsChange');
    emit('editPlc', plc, true);
}

function addTag(plc: PlcLinkSettings) {
    const plcTags = tagsFor(plc.name);
    if (plcTags.length >= MAX_TAGS_PER_PLC_LINK) {
        showMessage(
            t('dialog.plc_tag_config'),
            t('prompt.max_tags_per_plc_exceeded', [plc.name, MAX_TAGS_PER_PLC_LINK]),
        );
        return;
    }
    const lastTag = plcTags[plcTags.length - 1];
    const tag: TagSettings = lastTag
        ? {
              ...lastTag,
              id: newId(),
              name: `Tag ${props.tags.length + 1}`,
              plcLink: plc.name,
              color: PALETTE[props.tags.length % PALETTE.length],
              enabled: true,
          }
        : {
              id: newId(),
              name: `Tag ${props.tags.length + 1}`,
              plcLink: plc.name,
              address: 'DB1.DBD0',
              dataType: 'Real',
              yAxis: state.settings.yAxes[0]?.name ?? 'Y-Axis 1',
              color: PALETTE[props.tags.length % PALETTE.length],
              enabled: true,
              samplingIntervalMs: 0,
          };
    props.tags.push(tag);
    selectedPlcName.value = plc.name;
    selectedTagId.value = tag.id;
    expandedPlcs.value = new Set([...expandedPlcs.value, plc.name]);
    emit('settingsChange');
    emit('editTag', tag, true);
}

function toggleTag(tag: TagSettings, event: Event) {
    tag.enabled = (event.target as HTMLInputElement).checked;
    selectedTagId.value = tag.id;
    emit('settingsChange');
}

async function removePlc(plc: PlcLinkSettings) {
    if (props.plcLinks.length <= MIN_PLC_LINKS) {
        await showMessage(
            t('dialog.remove_plc'),
            t('prompt.min_plcs_required', [MIN_PLC_LINKS]),
        );
        return;
    }
    const index = props.plcLinks.indexOf(plc);
    if (index < 0) return;
    if (
        !(await showMessage(
            t('dialog.remove_plc'),
            t('prompt.remove_plc_confirm', [plc.name]),
            true,
        ))
    ) {
        return;
    }
    props.plcLinks.splice(index, 1);
    state.settings.tags = props.tags.filter((tag) => tag.plcLink !== plc.name);
    selectedPlcName.value = props.plcLinks[0]?.name ?? '';
    emit('settingsChange');
}

async function removeTag(tag: TagSettings) {
    const index = props.tags.indexOf(tag);
    if (index < 0) return;
    if (
        !(await showMessage(
            t('dialog.remove_tag'),
            t('prompt.remove_tag_confirm', [tag.name]),
            true,
        ))
    ) {
        return;
    }
    props.tags.splice(index, 1);
    selectedTagId.value = '';
    emit('settingsChange');
}

</script>

<style scoped>
.plc-tree-panel {
    flex: 0 0 250px;
    width: 250px;
    min-width: 220px;
    max-width: 420px;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: #ffffff;
    border: 1px solid var(--border-color);
    border-radius: 4px;
}

.tree-panel-header {
    flex: 0 0 32px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 0 6px 0 10px;
    background: var(--toolbar-bg);
    border-bottom: 1px solid var(--border-color);
}

.tree-panel-title {
    min-width: 0;
    overflow: hidden;
    color: var(--text-color);
    font-size: 12px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.tree-header-actions {
    display: flex;
    flex: 0 0 auto;
    gap: 2px;
}

.tree-icon-button {
    width: 28px;
    height: 28px;
    padding: 0;
    border: 1px solid transparent;
    border-radius: 3px;
    background: transparent;
    color: var(--text-color);
    cursor: pointer;
    line-height: 1;
}

.tree-icon-button .icon-glyph {
    display: block;
    width: 19px;
    height: 19px;
    margin: auto;
}

.tree-icon-button:disabled {
    opacity: 0.35;
    cursor: not-allowed;
}

.tree-icon-button:hover {
    border-color: var(--border-color);
    background: #ffffff;
}

.tree-scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 4px 3px;
}

.tree-list,
.tree-children {
    margin: 0;
    padding: 0;
    list-style: none;
}

.tree-children {
    padding-left: 18px;
}

.tree-node {
    min-height: 28px;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 2px 3px;
    border-radius: 3px;
}

.tree-node:hover,
.tree-node.selected {
    background: #eef5fb;
}

.tree-toggle {
    width: 22px;
    height: 26px;
    flex: 0 0 22px;
    padding: 0;
    border: 0;
    background: transparent;
    color: #64748b;
    cursor: pointer;
    line-height: 1;
}

.tree-toggle .icon-glyph {
    display: block;
    width: 18px;
    height: 18px;
    margin: auto;
}

.tree-toggle:hover {
    color: var(--accent);
}

.tree-status-dot {
    width: 8px;
    height: 8px;
    flex: 0 0 8px;
}

.tree-label {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    padding: 3px 2px;
    border: 0;
    background: transparent;
    color: var(--text-color);
    cursor: pointer;
    font: inherit;
    font-size: 12px;
    text-align: left;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.tree-label:focus {
    outline: none;
}

.tree-label:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: -2px;
}

.tree-plc-label {
    font-weight: 600;
}

.tree-tag-label {
    font-size: 12px;
}

.tree-node-meta {
    flex: 0 0 auto;
    max-width: 62px;
    overflow: hidden;
    color: #64748b;
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.tree-plc-node .tree-node-meta {
    font-size: 12px;
}

.tree-node-action {
    width: 22px;
    height: 26px;
    flex: 0 0 22px;
    padding: 0;
    border: 0;
    background: transparent;
    color: #64748b;
    cursor: pointer;
    line-height: 1;
    opacity: 0;
}

.tree-node-action .icon-glyph {
    display: block;
    width: 18px;
    height: 18px;
    margin: auto;
}

.tree-node:hover .tree-node-action,
.tree-node.selected .tree-node-action,
.tree-node-action:focus {
    opacity: 1;
}

.tree-node-action:hover {
    color: #dc2626;
}

.tree-add-tag-action:hover {
    color: var(--accent);
}

.tree-visibility {
    flex: 0 0 auto;
    margin: 0 2px;
    cursor: pointer;
}

.tree-color-dot {
    width: 8px;
    height: 8px;
    flex: 0 0 8px;
    border-radius: 50%;
}

.tree-empty,
.tree-child-empty {
    padding: 12px 8px;
    color: #64748b;
    font-size: 11px;
    font-style: italic;
    text-align: center;
}

.tree-child-empty {
    padding: 5px 8px;
    text-align: left;
}
</style>
