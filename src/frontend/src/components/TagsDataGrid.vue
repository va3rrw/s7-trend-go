<template>
    <div
        id="tagsArea"
        class="tags-area">
        <div class="tags-area-toolbar">
            <div class="tags-area-title">
                <span class="tags-count">{{
                    $t('grid.tags_count', [
                        sortedTags.length,
                        tags.length,
                    ])
                }}</span>
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
            <table
                class="datagrid"
                id="gridTags">
                <thead>
                    <tr>
                        <!-- Col 1 headers -->
                        <th class="col-checkbox no-sort">
                            <input
                                type="checkbox"
                                :checked="allTagsEnabled"
                                :title="$t('grid.on')"
                                @change="toggleAllTags" />
                        </th>
                        <th
                            class="col-text"
                            @click="toggleSort('name')">
                            {{ $t('grid.tag') }}
                            <span
                                v-if="sortKey === 'name'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-text"
                            @click="toggleSort('plcLink')">
                            {{ $t('grid.plc') }}
                            <span
                                v-if="sortKey === 'plcLink'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('value')">
                            {{ $t('grid.value') }}
                            <span
                                v-if="sortKey === 'value'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('min')">
                            {{ $t('grid.min') }}
                            <span
                                v-if="sortKey === 'min'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('max')">
                            {{ $t('grid.max') }}
                            <span
                                v-if="sortKey === 'max'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>

                        <!-- Center Divider 1 -->
                        <th class="col-divider no-sort" />

                        <!-- Col 2 headers -->
                        <th class="col-checkbox no-sort">
                            <input
                                type="checkbox"
                                :checked="allTagsEnabled"
                                :title="$t('grid.on')"
                                @change="toggleAllTags" />
                        </th>
                        <th
                            class="col-text"
                            @click="toggleSort('name')">
                            {{ $t('grid.tag') }}
                            <span
                                v-if="sortKey === 'name'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-text"
                            @click="toggleSort('plcLink')">
                            {{ $t('grid.plc') }}
                            <span
                                v-if="sortKey === 'plcLink'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('value')">
                            {{ $t('grid.value') }}
                            <span
                                v-if="sortKey === 'value'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('min')">
                            {{ $t('grid.min') }}
                            <span
                                v-if="sortKey === 'min'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('max')">
                            {{ $t('grid.max') }}
                            <span
                                v-if="sortKey === 'max'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>

                        <!-- Center Divider 2 -->
                        <th class="col-divider no-sort" />

                        <!-- Col 3 headers -->
                        <th class="col-checkbox no-sort">
                            <input
                                type="checkbox"
                                :checked="allTagsEnabled"
                                :title="$t('grid.on')"
                                @change="toggleAllTags" />
                        </th>
                        <th
                            class="col-text"
                            @click="toggleSort('name')">
                            {{ $t('grid.tag') }}
                            <span
                                v-if="sortKey === 'name'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-text"
                            @click="toggleSort('plcLink')">
                            {{ $t('grid.plc') }}
                            <span
                                v-if="sortKey === 'plcLink'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('value')">
                            {{ $t('grid.value') }}
                            <span
                                v-if="sortKey === 'value'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('min')">
                            {{ $t('grid.min') }}
                            <span
                                v-if="sortKey === 'min'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                        <th
                            class="col-numeric"
                            @click="toggleSort('max')">
                            {{ $t('grid.max') }}
                            <span
                                v-if="sortKey === 'max'"
                                class="sort-icon"
                                >{{ sortAsc ? '▲' : '▼' }}</span
                            >
                        </th>
                    </tr>
                </thead>
                <tbody>
                    <tr
                        v-if="tripletTags.length === 0"
                        class="empty-row">
                        <td
                            colspan="20"
                            class="datagrid-empty">
                            {{
                                tags.length === 0
                                    ? $t('grid.no_tags')
                                    : $t('grid.no_matching_tags')
                            }}
                        </td>
                    </tr>
                    <tr
                        v-for="triplet in tripletTags"
                        :key="triplet.c1.id">
                        <!-- Col 1 Cell Group -->
                        <td
                            class="col-checkbox"
                            :style="{
                                backgroundColor:
                                    triplet.c1.color || 'transparent',
                            }"
                            @click.stop>
                            <input
                                type="checkbox"
                                :checked="triplet.c1.enabled"
                                @change="
                                    triplet.c1.enabled = (
                                        $event.target as HTMLInputElement
                                    ).checked;
                                    emit('settingsChange');
                                " />
                        </td>
                        <td
                            class="col-text font-medium"
                            :style="{
                                backgroundColor:
                                    triplet.c1.color || 'transparent',
                            }"
                            @dblclick="emit('editTag', triplet.c1)">
                            {{ triplet.c1.name }}
                        </td>
                        <td
                            class="col-text"
                            :style="{
                                backgroundColor:
                                    triplet.c1.color || 'transparent',
                            }"
                            @dblclick="emit('editTag', triplet.c1)">
                            {{ triplet.c1.plcLink }}
                        </td>
                        <td
                            class="col-numeric font-mono"
                            :style="{
                                backgroundColor:
                                    triplet.c1.color || 'transparent',
                            }"
                            @dblclick="emit('editTag', triplet.c1)">
                            {{ tagValue(triplet.c1.id) }}
                        </td>
                        <td
                            class="col-numeric font-mono"
                            :style="{
                                backgroundColor:
                                    triplet.c1.color || 'transparent',
                            }"
                            @dblclick="emit('editTag', triplet.c1)">
                            {{ tagMin(triplet.c1.id) }}
                        </td>
                        <td
                            class="col-numeric font-mono"
                            :style="{
                                backgroundColor:
                                    triplet.c1.color || 'transparent',
                            }"
                            @dblclick="emit('editTag', triplet.c1)">
                            {{ tagMax(triplet.c1.id) }}
                        </td>

                        <!-- Divider 1 -->
                        <td class="col-divider" />

                        <!-- Col 2 Cell Group -->
                        <template v-if="triplet.c2">
                            <td
                                class="col-checkbox"
                                :style="{
                                    backgroundColor:
                                        triplet.c2?.color ||
                                        'transparent',
                                }"
                                @click.stop>
                                <input
                                    type="checkbox"
                                    :checked="triplet.c2?.enabled"
                                    @change="
                                        if (triplet.c2) {
                                            triplet.c2.enabled = (
                                                $event.target as HTMLInputElement
                                            ).checked;
                                            emit('settingsChange');
                                        }
                                    " />
                            </td>
                            <td
                                class="col-text font-medium"
                                :style="{
                                    backgroundColor:
                                        triplet.c2?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c2 &&
                                    emit('editTag', triplet.c2)
                                ">
                                {{ triplet.c2?.name }}
                            </td>
                            <td
                                class="col-text"
                                :style="{
                                    backgroundColor:
                                        triplet.c2?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c2 &&
                                    emit('editTag', triplet.c2)
                                ">
                                {{ triplet.c2?.plcLink }}
                            </td>
                            <td
                                class="col-numeric font-mono"
                                :style="{
                                    backgroundColor:
                                        triplet.c2?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c2 &&
                                    emit('editTag', triplet.c2)
                                ">
                                {{
                                    triplet.c2
                                        ? tagValue(triplet.c2.id)
                                        : '-'
                                }}
                            </td>
                            <td
                                class="col-numeric font-mono"
                                :style="{
                                    backgroundColor:
                                        triplet.c2?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c2 &&
                                    emit('editTag', triplet.c2)
                                ">
                                {{
                                    triplet.c2
                                        ? tagMin(triplet.c2.id)
                                        : '-'
                                }}
                            </td>
                            <td
                                class="col-numeric font-mono"
                                :style="{
                                    backgroundColor:
                                        triplet.c2?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c2 &&
                                    emit('editTag', triplet.c2)
                                ">
                                {{
                                    triplet.c2
                                        ? tagMax(triplet.c2.id)
                                        : '-'
                                }}
                            </td>
                        </template>
                        <template v-else>
                            <td
                                colspan="6"
                                class="empty-cell" />
                        </template>

                        <!-- Divider 2 -->
                        <td class="col-divider" />

                        <!-- Col 3 Cell Group -->
                        <template v-if="triplet.c3">
                            <td
                                class="col-checkbox"
                                :style="{
                                    backgroundColor:
                                        triplet.c3?.color ||
                                        'transparent',
                                }"
                                @click.stop>
                                <input
                                    type="checkbox"
                                    :checked="triplet.c3?.enabled"
                                    @change="
                                        if (triplet.c3) {
                                            triplet.c3.enabled = (
                                                $event.target as HTMLInputElement
                                            ).checked;
                                            emit('settingsChange');
                                        }
                                    " />
                            </td>
                            <td
                                class="col-text font-medium"
                                :style="{
                                    backgroundColor:
                                        triplet.c3?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c3 &&
                                    emit('editTag', triplet.c3)
                                ">
                                {{ triplet.c3?.name }}
                            </td>
                            <td
                                class="col-text"
                                :style="{
                                    backgroundColor:
                                        triplet.c3?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c3 &&
                                    emit('editTag', triplet.c3)
                                ">
                                {{ triplet.c3?.plcLink }}
                            </td>
                            <td
                                class="col-numeric font-mono"
                                :style="{
                                    backgroundColor:
                                        triplet.c3?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c3 &&
                                    emit('editTag', triplet.c3)
                                ">
                                {{
                                    triplet.c3
                                        ? tagValue(triplet.c3.id)
                                        : '-'
                                }}
                            </td>
                            <td
                                class="col-numeric font-mono"
                                :style="{
                                    backgroundColor:
                                        triplet.c3?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c3 &&
                                    emit('editTag', triplet.c3)
                                ">
                                {{
                                    triplet.c3
                                        ? tagMin(triplet.c3.id)
                                        : '-'
                                }}
                            </td>
                            <td
                                class="col-numeric font-mono"
                                :style="{
                                    backgroundColor:
                                        triplet.c3?.color ||
                                        'transparent',
                                }"
                                @dblclick="
                                    triplet.c3 &&
                                    emit('editTag', triplet.c3)
                                ">
                                {{
                                    triplet.c3
                                        ? tagMax(triplet.c3.id)
                                        : '-'
                                }}
                            </td>
                        </template>
                        <template v-else>
                            <td
                                colspan="6"
                                class="empty-cell" />
                        </template>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { TagSettings } from '../types';
import { formatNumber } from '../store';

const props = defineProps<{
    tags: TagSettings[];
    liveValues: Record<string, string>;
    sampledRange: Record<string, { min: number; max: number }>;
}>();

const emit = defineEmits<{
    editTag: [tag: TagSettings];
    settingsChange: [];
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
        if (sortAsc.value) {
            sortAsc.value = false;
        } else {
            sortKey.value = null;
            sortAsc.value = true;
        }
    } else {
        sortKey.value = key;
        sortAsc.value = true;
    }
}

const allTagsEnabled = computed(() => {
    if (!props.tags.length) return false;
    return props.tags.every((t) => t.enabled);
});

function toggleAllTags(e: Event) {
    const checked = (e.target as HTMLInputElement).checked;
    props.tags.forEach((t) => {
        t.enabled = checked;
    });
    emit('settingsChange');
}

const sortedTags = computed(() => {
    let list = props.tags;
    const q = searchQuery.value.trim().toLowerCase();
    if (q) {
        list = list.filter(
            (t) =>
                t.name.toLowerCase().includes(q) ||
                t.plcLink.toLowerCase().includes(q) ||
                t.address.toLowerCase().includes(q) ||
                t.dataType.toLowerCase().includes(q) ||
                (t.yAxis && t.yAxis.toLowerCase().includes(q)),
        );
    }

    if (!sortKey.value) return list;

    const k = sortKey.value;
    const dir = sortAsc.value ? 1 : -1;

    return [...list].sort((a, b) => {
        if (k === 'name') return a.name.localeCompare(b.name) * dir;
        if (k === 'plcLink') return a.plcLink.localeCompare(b.plcLink) * dir;
        if (k === 'value') {
            const vA = parseFloat(props.liveValues[a.id] ?? '');
            const vB = parseFloat(props.liveValues[b.id] ?? '');
            if (Number.isFinite(vA) && Number.isFinite(vB))
                return (vA - vB) * dir;
            return (
                (props.liveValues[a.id] || '').localeCompare(
                    props.liveValues[b.id] || '',
                ) * dir
            );
        }
        if (k === 'min') {
            const mA =
                props.sampledRange[a.id]?.min ??
                (sortAsc.value ? Infinity : -Infinity);
            const mB =
                props.sampledRange[b.id]?.min ??
                (sortAsc.value ? Infinity : -Infinity);
            return (mA - mB) * dir;
        }
        if (k === 'max') {
            const mA =
                props.sampledRange[a.id]?.max ??
                (sortAsc.value ? -Infinity : Infinity);
            const mB =
                props.sampledRange[b.id]?.max ??
                (sortAsc.value ? -Infinity : Infinity);
            return (mA - mB) * dir;
        }
        return 0;
    });
});

const tripletTags = computed<TagTriplet[]>(() => {
    const list = sortedTags.value;
    const triplets: TagTriplet[] = [];
    const r = Math.ceil(list.length / 3);
    for (let i = 0; i < r; i++) {
        triplets.push({
            c1: list[i],
            c2: list[i + r],
            c3: list[i + 2 * r],
        });
    }
    return triplets;
});

function tagValue(id: string) {
    return props.liveValues[id] ?? '-';
}
function tagMin(id: string) {
    const r = props.sampledRange[id];
    return r ? formatNumber(r.min) : '-';
}
function tagMax(id: string) {
    const r = props.sampledRange[id];
    return r ? formatNumber(r.max) : '-';
}
</script>
