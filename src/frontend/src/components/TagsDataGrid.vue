<template>
    <div id="tagsArea" class="tags-area">
        <div class="tags-area-toolbar">
            <div class="tags-area-title">
                <span class="tags-count">
                    {{ $t('grid.tags_count', [sortedTags.length, tags.length]) }}
                </span>
            </div>
            <div class="tags-search-wrapper">
                <input
                    v-model="searchQuery"
                    type="text"
                    :placeholder="$t('grid.search')"
                    class="tags-search-input" />
                <button
                    v-if="searchQuery"
                    type="button"
                    class="tags-search-clear"
                    @click="searchQuery = ''">
                    ✕
                </button>
            </div>
        </div>

        <div class="tags-table-container">
            <table id="gridTags" class="datagrid">
                <colgroup>
                    <template v-for="group in 3" :key="`col-${group}`">
                        <col class="col-tag" />
                        <col class="col-plc" />
                        <col class="col-value" />
                        <col class="col-min" />
                        <col class="col-max" />
                        <col v-if="group < 3" class="col-divider" />
                    </template>
                </colgroup>
                <thead>
                    <tr>
                        <template v-for="group in 3" :key="`header-${group}`">
                            <th class="col-tag col-text" @click="toggleSort('name')">
                                {{ $t('grid.tag') }}
                                <span v-if="sortKey === 'name'" class="sort-icon">
                                    {{ sortAsc ? '▲' : '▼' }}
                                </span>
                            </th>
                            <th class="col-plc col-text" @click="toggleSort('plcLink')">
                                {{ $t('grid.plc') }}
                                <span v-if="sortKey === 'plcLink'" class="sort-icon">
                                    {{ sortAsc ? '▲' : '▼' }}
                                </span>
                            </th>
                            <th class="col-value col-numeric" @click="toggleSort('value')">
                                {{ $t('grid.value') }}
                                <span v-if="sortKey === 'value'" class="sort-icon">
                                    {{ sortAsc ? '▲' : '▼' }}
                                </span>
                            </th>
                            <th class="col-min col-numeric" @click="toggleSort('min')">
                                {{ $t('grid.min') }}
                                <span v-if="sortKey === 'min'" class="sort-icon">
                                    {{ sortAsc ? '▲' : '▼' }}
                                </span>
                            </th>
                            <th class="col-max col-numeric" @click="toggleSort('max')">
                                {{ $t('grid.max') }}
                                <span v-if="sortKey === 'max'" class="sort-icon">
                                    {{ sortAsc ? '▲' : '▼' }}
                                </span>
                            </th>
                            <th v-if="group < 3" class="col-divider no-sort" />
                        </template>
                    </tr>
                </thead>
                <tbody>
                    <tr v-if="tripletTags.length === 0" class="empty-row">
                        <td colspan="17" class="datagrid-empty">
                            {{ tags.length === 0 ? $t('grid.no_tags') : $t('grid.no_matching_tags') }}
                        </td>
                    </tr>
                    <tr v-for="triplet in tripletTags" :key="triplet.c1.id">
                        <template v-for="(tag, index) in tripletItems(triplet)" :key="`${triplet.c1.id}-${index}`">
                            <template v-if="tag">
                                <td
                                    class="col-tag col-text font-medium"
                                    :style="cellStyle(tag)"
                                    @dblclick="emit('editTag', tag)">
                                    {{ tag.name }}
                                </td>
                                <td
                                    class="col-plc col-text"
                                    :style="cellStyle(tag)"
                                    @dblclick="emit('editTag', tag)">
                                    {{ tag.plcLink }}
                                </td>
                                <td
                                    class="col-value col-numeric font-mono"
                                    :style="cellStyle(tag)"
                                    @dblclick="emit('editTag', tag)">
                                    {{ tagValue(tag.id) }}
                                </td>
                                <td
                                    class="col-min col-numeric font-mono"
                                    :style="cellStyle(tag)"
                                    @dblclick="emit('editTag', tag)">
                                    {{ tagMin(tag.id) }}
                                </td>
                                <td
                                    class="col-max col-numeric font-mono"
                                    :style="cellStyle(tag)"
                                    @dblclick="emit('editTag', tag)">
                                    {{ tagMax(tag.id) }}
                                </td>
                            </template>
                            <td v-else colspan="5" class="empty-cell" />
                            <td v-if="index < 2" class="col-divider" />
                        </template>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { TagSettings } from '../types';
import { formatNumber } from '../store';

const props = defineProps<{
    tags: TagSettings[];
    liveValues: Record<string, string>;
    sampledRange: Record<string, { min: number; max: number }>;
}>();

const emit = defineEmits<{
    editTag: [tag: TagSettings];
}>();

type SortKey = 'name' | 'plcLink' | 'value' | 'min' | 'max';

interface TagTriplet {
    c1: TagSettings;
    c2?: TagSettings;
    c3?: TagSettings;
}

const sortKey = ref<SortKey | null>(null);
const sortAsc = ref(true);
const searchQuery = ref('');

function toggleSort(key: SortKey) {
    if (sortKey.value === key) {
        if (sortAsc.value) sortAsc.value = false;
        else {
            sortKey.value = null;
            sortAsc.value = true;
        }
    } else {
        sortKey.value = key;
        sortAsc.value = true;
    }
}

const sortedTags = computed(() => {
    let list = props.tags;
    const query = searchQuery.value.trim().toLowerCase();
    if (query) {
        list = list.filter(
            (tag) =>
                tag.name.toLowerCase().includes(query) ||
                tag.plcLink.toLowerCase().includes(query) ||
                tag.address.toLowerCase().includes(query) ||
                tag.dataType.toLowerCase().includes(query) ||
                (tag.yAxis && tag.yAxis.toLowerCase().includes(query)),
        );
    }

    if (!sortKey.value) return list;
    const key = sortKey.value;
    const direction = sortAsc.value ? 1 : -1;

    return [...list].sort((a, b) => {
        if (key === 'name') return a.name.localeCompare(b.name) * direction;
        if (key === 'plcLink') return a.plcLink.localeCompare(b.plcLink) * direction;
        if (key === 'value') {
            const valueA = parseFloat(props.liveValues[a.id] ?? '');
            const valueB = parseFloat(props.liveValues[b.id] ?? '');
            if (Number.isFinite(valueA) && Number.isFinite(valueB)) {
                return (valueA - valueB) * direction;
            }
            return (props.liveValues[a.id] || '').localeCompare(props.liveValues[b.id] || '') * direction;
        }
        if (key === 'min') {
            const minA = props.sampledRange[a.id]?.min ?? (sortAsc.value ? Infinity : -Infinity);
            const minB = props.sampledRange[b.id]?.min ?? (sortAsc.value ? Infinity : -Infinity);
            return (minA - minB) * direction;
        }
        const maxA = props.sampledRange[a.id]?.max ?? (sortAsc.value ? -Infinity : Infinity);
        const maxB = props.sampledRange[b.id]?.max ?? (sortAsc.value ? -Infinity : Infinity);
        return (maxA - maxB) * direction;
    });
});

const tripletTags = computed<TagTriplet[]>(() => {
    const result: TagTriplet[] = [];
    const rows = Math.ceil(sortedTags.value.length / 3);
    for (let index = 0; index < rows; index++) {
        result.push({
            c1: sortedTags.value[index],
            c2: sortedTags.value[index + rows],
            c3: sortedTags.value[index + rows * 2],
        });
    }
    return result;
});

function tripletItems(triplet: TagTriplet) {
    return [triplet.c1, triplet.c2, triplet.c3];
}

function cellStyle(tag: TagSettings) {
    return { backgroundColor: tag.color || 'transparent' };
}

function tagValue(id: string) {
    return props.liveValues[id] ?? '-';
}

function tagMin(id: string) {
    const range = props.sampledRange[id];
    return range ? formatNumber(range.min) : '-';
}

function tagMax(id: string) {
    const range = props.sampledRange[id];
    return range ? formatNumber(range.max) : '-';
}
</script>
