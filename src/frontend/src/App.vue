<template>
    <div id="app">
        <!-- Menu Bar — layout mirrors old S7TrendValues (File / Edit / Tools / Trace / Help) -->
        <nav class="menubar">
            <div class="menu-item">
                <span>{{ $t('menu.file') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="menuLoadSettings">
                        <span>{{ $t('menu.load_settings') }}</span>
                        <span class="menu-shortcut">Ctrl+L</span>
                    </a>
                    <a
                        href="#"
                        @click.prevent="menuSaveSettings">
                        <span>{{ $t('menu.save_settings') }}</span>
                        <span class="menu-shortcut">Ctrl+S</span>
                    </a>
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.edit') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="plcTagsOpen = true">
                        <span>{{ $t('menu.plc_tag_settings') }}</span>
                        <span class="menu-shortcut">Ctrl+T</span>
                    </a>
                    <a
                        href="#"
                        @click.prevent="menuTrendWindow">
                        <span>{{ $t('menu.x_settings') }}</span>
                        <span class="menu-shortcut">Ctrl+X</span>
                    </a>
                    <a
                        href="#"
                        @click.prevent="yAxesOpen = true">
                        <span>{{ $t('menu.y_settings') }}</span>
                        <span class="menu-shortcut">Ctrl+Y</span>
                    </a>
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.tools') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="toggleCursors">
                        <span
                            >{{ cursorsEnabled ? '✓ ' : ''
                            }}{{ $t('menu.measurement_cursors') }}</span
                        >
                        <span class="menu-shortcut">Ctrl+M</span>
                    </a>
                    <hr />
                    <a
                        href="#"
                        @click.prevent="menuExport">
                        <span>{{ $t('menu.export_csv') }}</span>
                    </a>
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.trace') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="menuTimer">
                        <span>{{ $t('menu.set_timer_interval') }}</span>
                    </a>
                    <hr />
                    <a
                        href="#"
                        :class="{ disabled: state.isSampling }"
                        @click.prevent="!state.isSampling && startSampling()">
                        <span>{{ $t('menu.start') }}</span>
                        <span class="menu-shortcut">F5</span>
                    </a>
                    <a
                        href="#"
                        :class="{ disabled: !state.isSampling }"
                        @click.prevent="state.isSampling && pauseSampling()">
                        <span>{{
                            state.isPaused
                                ? $t('menu.resume')
                                : $t('menu.pause')
                        }}</span>
                        <span class="menu-shortcut">Space</span>
                    </a>
                    <a
                        href="#"
                        :class="{ disabled: !state.isSampling }"
                        @click.prevent="state.isSampling && stopSampling()">
                        <span>{{ $t('menu.stop') }}</span>
                        <span class="menu-shortcut">Shift+F5</span>
                    </a>
                    <hr />
                    <a
                        href="#"
                        @click.prevent="menuInterpolation">
                        <span>{{ $t('menu.samples_interpolation') }}</span>
                    </a>
                    <hr />
                    <a
                        href="#"
                        @click.prevent="menuClear">
                        <span>{{ $t('menu.clear_records') }}</span>
                    </a>
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.help') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="locale = 'en'">
                        <span>English</span>
                    </a>
                    <a
                        href="#"
                        @click.prevent="locale = 'zh'">
                        <span>简体中文</span>
                    </a>
                    <hr />
                    <a
                        href="#"
                        @click.prevent="aboutOpen = true">
                        <span>{{ $t('menu.about') }}</span>
                    </a>
                </div>
            </div>
        </nav>

        <!-- Toolbar -->
        <div class="toolbar">
            <button
                class="tool-btn"
                :title="$t('toolbar.plc_tag_settings')"
                @click="plcTagsOpen = true">
                <span
                    class="i-ix-plc-tag icon-glyph"
                    aria-hidden="true" />
            </button>
            <div class="separator" />
            <button
                class="tool-btn"
                :title="$t('toolbar.y_axis_settings')"
                @click="yAxesOpen = true">
                <span
                    class="i-ix-y-axis-settings icon-glyph"
                    aria-hidden="true" />
            </button>
            <button
                class="tool-btn"
                :title="$t('toolbar.x_axis_settings')"
                @click="menuTrendWindow">
                <span
                    class="i-ix-x-axis-settings icon-glyph"
                    aria-hidden="true" />
            </button>
            <div class="separator" />
            <button
                class="tool-btn"
                :class="{ active: cursorsEnabled }"
                :title="$t('toolbar.cursors')"
                @click="toggleCursors">
                <span
                    class="i-ix-ruler-horizontal icon-glyph"
                    aria-hidden="true" />
            </button>
            <div class="separator" />
            <button
                class="tool-btn start"
                :title="$t('toolbar.start_sampling')"
                :disabled="state.isSampling"
                @click="startSampling">
                <span
                    class="i-ix-play icon-glyph"
                    aria-hidden="true" />
            </button>
            <button
                class="tool-btn pause"
                :title="
                    state.isPaused
                        ? $t('toolbar.resume_chart')
                        : $t('toolbar.pause_chart')
                "
                :disabled="!state.isSampling"
                @click="pauseSampling">
                <span
                    :class="[
                        'icon-glyph',
                        state.isPaused ? 'i-ix-play-pause' : 'i-ix-pause',
                    ]"
                    aria-hidden="true" />
            </button>
            <button
                class="tool-btn stop"
                :title="$t('toolbar.stop_sampling')"
                :disabled="!state.isSampling"
                @click="stopSampling">
                <span
                    class="i-ix-stop icon-glyph"
                    aria-hidden="true" />
            </button>
        </div>

        <!-- Main Content Area -->
        <div class="main-content">
            <TrendChartView
                ref="chartRef"
                :tags="chartTags"
                :axes="chartAxes"
                :time-window-seconds="state.settings.timeWindowSeconds"
                :interpolation="state.settings.interpolation"
                :cursors-enabled="cursorsEnabled"
                @drag-start="onChartDragStart"
                @cursor-change="onCursorChange" />

            <!-- Measurement & Statistics Strip -->
            <MeasurementStrip
                :open="cursorsEnabled"
                :measurement="measurementData"
                :tags="state.settings.tags"
                @close="cursorsEnabled = false"
                @fit-window="fitCursors" />

            <!-- Resizer -->
            <div
                class="resizer"
                @pointerdown="onResizerPointerDown" />

            <!-- Tags DataGrid Area -->
            <div
                id="tagsArea"
                class="tags-area">
                <div class="tags-area-toolbar">
                    <div class="tags-area-title">
                        <span class="tags-count">{{
                            $t('grid.tags_count', [
                                sortedTags.length,
                                state.settings.tags.length,
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
                                        state.settings.tags.length === 0
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
                                            onSettingsChanged();
                                        " />
                                </td>
                                <td
                                    class="col-text font-medium"
                                    :style="{
                                        backgroundColor:
                                            triplet.c1.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(triplet.c1)">
                                    {{ triplet.c1.name }}
                                </td>
                                <td
                                    class="col-text"
                                    :style="{
                                        backgroundColor:
                                            triplet.c1.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(triplet.c1)">
                                    {{ triplet.c1.plcLink }}
                                </td>
                                <td
                                    class="col-numeric font-mono"
                                    :style="{
                                        backgroundColor:
                                            triplet.c1.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(triplet.c1)">
                                    {{ tagValue(triplet.c1.id) }}
                                </td>
                                <td
                                    class="col-numeric font-mono"
                                    :style="{
                                        backgroundColor:
                                            triplet.c1.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(triplet.c1)">
                                    {{ tagMin(triplet.c1.id) }}
                                </td>
                                <td
                                    class="col-numeric font-mono"
                                    :style="{
                                        backgroundColor:
                                            triplet.c1.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(triplet.c1)">
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
                                                    onSettingsChanged();
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
                                            onRowDblClick(triplet.c2)
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
                                            onRowDblClick(triplet.c2)
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
                                            onRowDblClick(triplet.c2)
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
                                            onRowDblClick(triplet.c2)
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
                                            onRowDblClick(triplet.c2)
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
                                                    onSettingsChanged();
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
                                            onRowDblClick(triplet.c3)
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
                                            onRowDblClick(triplet.c3)
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
                                            onRowDblClick(triplet.c3)
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
                                            onRowDblClick(triplet.c3)
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
                                            onRowDblClick(triplet.c3)
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
        </div>

        <!-- Status Bar -->
        <footer class="statusbar">
            <div class="status-item">{{ state.statusMessage }}</div>
            <div class="status-sep" />
            <div class="status-item">
                {{ $t('status.window') }}
                {{ state.settings.timeWindowSeconds }} s
            </div>
            <div class="status-sep" />
            <div class="status-item">
                {{ $t('status.interval') }}
                {{ state.settings.pollIntervalMs }} ms
            </div>
            <div class="status-sep" />
            <div class="status-item">
                {{ $t('status.actual') }}
                {{ actualSamplingText }}
            </div>
        </footer>
    </div>

    <!-- Dialogs -->
    <PlcTagsDialog
        :open="plcTagsOpen"
        @close="plcTagsOpen = false"
        @save="onPlcTagsSave" />
    <YAxesDialog
        :open="yAxesOpen"
        @close="yAxesOpen = false"
        @save="onYAxesSave" />
    <AboutDialog
        :open="aboutOpen"
        @close="aboutOpen = false" />

    <PromptDialog
        :open="promptOpen"
        :title="promptTitle"
        :text="promptText"
        :default-value="promptDefault"
        :options="promptOptions"
        @close="onPromptClose" />

    <MessageDialog
        :open="uiState.msgOpen"
        :title="uiState.msgTitle"
        :message="uiState.msgMessage"
        :show-cancel="uiState.msgShowCancel"
        @close="onMsgClose" />

    <!-- Double-click tag editor (from main grid) -->
    <TagEditorDialog
        :open="dblEditOpen"
        :tag="dblEditTag"
        :index="dblEditIndex"
        :existing-tags="state.settings.tags"
        @save="onDblEditSave"
        @close="dblEditOpen = false" />
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import type { ChartTag, ChartAxis, CursorMeasurement } from './chart';
import type { TagSettings } from './types';
import {
    state,
    backend,
    wailsRuntime,
    updateTagValue,
    markPlcForTag,
    getTag,
    resetStats,
    formatNumber,
    uiState,
    showMessage,
} from './store';
import TrendChartView from './components/TrendChartView.vue';
import MeasurementStrip from './components/MeasurementStrip.vue';
import PlcTagsDialog from './components/PlcTagsDialog.vue';
import YAxesDialog from './components/YAxesDialog.vue';
import PromptDialog from './components/PromptDialog.vue';
import MessageDialog from './components/MessageDialog.vue';
import AboutDialog from './components/AboutDialog.vue';
import TagEditorDialog from './components/TagEditorDialog.vue';

const { t, locale } = useI18n();

// ── Chart reference ────────────────────────────────────────────────
const chartRef = ref<InstanceType<typeof TrendChartView> | null>(null);

// ── Cursors & Measurement ──────────────────────────────────────────
const cursorsEnabled = ref(false);
const measurementData = ref<CursorMeasurement | null>(null);

function toggleCursors() {
    cursorsEnabled.value = !cursorsEnabled.value;
}

function onCursorChange(m: CursorMeasurement | null) {
    measurementData.value = m;
}

function fitCursors() {
    chartRef.value?.fitCursorsToWindow();
}

function handleKeydown(e: KeyboardEvent) {
    const target = e.target as HTMLElement | null;
    const isInput =
        target &&
        (target.tagName === 'INPUT' ||
            target.tagName === 'TEXTAREA' ||
            target.tagName === 'SELECT' ||
            target.isContentEditable);

    const hasModalOpen =
        plcTagsOpen.value ||
        yAxesOpen.value ||
        aboutOpen.value ||
        promptOpen.value ||
        uiState.msgOpen ||
        dblEditOpen.value;

    // Ctrl + M / Cmd + M / Alt + M: Toggle measurement cursors
    if (
        (((e.ctrlKey || e.metaKey) && !e.altKey) ||
            (e.altKey && !e.ctrlKey && !e.metaKey)) &&
        (e.key === 'm' || e.key === 'M') &&
        !e.shiftKey
    ) {
        e.preventDefault();
        toggleCursors();
        return;
    }

    // Ctrl + S / Cmd + S: Save settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 's' || e.key === 'S') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        e.preventDefault();
        void menuSaveSettings();
        return;
    }

    // Ctrl + L / Ctrl + O / Cmd + L / Cmd + O: Load settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 'l' || e.key === 'L' || e.key === 'o' || e.key === 'O') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        e.preventDefault();
        void menuLoadSettings();
        return;
    }

    // Ctrl + X / Cmd + X: X-axis settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 'x' || e.key === 'X') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        if (!isInput && !hasModalOpen) {
            e.preventDefault();
            void menuTrendWindow();
            return;
        }
    }

    // Ctrl + Y / Cmd + Y: Y-axis settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 'y' || e.key === 'Y') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        if (!isInput && !hasModalOpen) {
            e.preventDefault();
            yAxesOpen.value = true;
            return;
        }
    }

    // Ctrl + T / Cmd + T: PLC / Tag settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 't' || e.key === 'T') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        if (!isInput && !hasModalOpen) {
            e.preventDefault();
            plcTagsOpen.value = true;
            return;
        }
    }

    // Shift + F5: Stop sampling
    if (e.shiftKey && e.key === 'F5') {
        e.preventDefault();
        void stopSampling();
        return;
    }

    // F5: Start sampling (or resume if paused)
    if (
        !e.shiftKey &&
        !e.ctrlKey &&
        !e.altKey &&
        !e.metaKey &&
        e.key === 'F5'
    ) {
        e.preventDefault();
        if (!state.isSampling) {
            void startSampling();
        } else if (state.isPaused) {
            pauseSampling();
        }
        return;
    }

    // Space: Toggle pause / resume (only when no modal is open and not typing inside an input)
    if (
        !e.ctrlKey &&
        !e.altKey &&
        !e.metaKey &&
        !e.shiftKey &&
        (e.key === ' ' || e.code === 'Space')
    ) {
        if (!isInput && !hasModalOpen && state.isSampling) {
            e.preventDefault();
            pauseSampling();
            return;
        }
    }
}

// ── Dialog visibility ──────────────────────────────────────────────
const plcTagsOpen = ref(false);
const yAxesOpen = ref(false);
const aboutOpen = ref(false);

// Prompt dialog
const promptOpen = ref(false);
const promptTitle = ref('');
const promptText = ref('');
const promptDefault = ref('');
const promptOptions = ref<string[]>();
let promptResolve: ((val: string | null) => void) | null = null;

function showPrompt(
    title: string,
    text: string,
    defaultValue: string,
    options?: string[],
): Promise<string | null> {
    return new Promise((resolve) => {
        promptTitle.value = title;
        promptText.value = text;
        promptDefault.value = defaultValue;
        promptOptions.value = options;
        promptResolve = resolve;
        promptOpen.value = true;
    });
}

function showConfirm(title: string, message: string): Promise<boolean> {
    return showMessage(title, message, true);
}

// Double-click edit on main grid – reuse TagEditorDialog directly
const dblEditOpen = ref(false);
const dblEditTag = ref<TagSettings | null>(null);
const dblEditIndex = ref(-1);

function onRowDblClick(tag: TagSettings) {
    const idx = state.settings.tags.findIndex((t) => t.id === tag.id);
    if (idx < 0) return;
    dblEditIndex.value = idx;
    dblEditTag.value = { ...tag };
    dblEditOpen.value = true;
}

function onDblEditSave(updatedTag: TagSettings) {
    if (
        dblEditIndex.value >= 0 &&
        dblEditIndex.value < state.settings.tags.length
    ) {
        state.settings.tags[dblEditIndex.value] = updatedTag;
        onSettingsChanged();
    }
    dblEditOpen.value = false;
}

// Actual polling interval tracking
const actualByPlc = ref(new Map<string, number>());
const actualSamplingText = computed(() => {
    if (!state.isSampling) return '-';
    const entries = Array.from(actualByPlc.value.entries());
    if (!entries.length) return '-';
    return entries.map(([link, ms]) => `${link}: ${ms}ms`).join(', ');
});

function resetActualSampling() {
    actualByPlc.value = new Map();
}

// Browser preview fallback timer (dev mode only)
let fallbackTimer: number | null = null;

// ── Settings sync ──────────────────────────────────────────────────
async function saveSettingsAndRestart() {
    try {
        const api = backend();
        if (state.isSampling) {
            if (api?.StartPolling) {
                await api.StartPolling(state.settings);
            }
        } else {
            if (api?.SaveSettings) {
                await api.SaveSettings(state.settings);
            }
        }
    } catch (err) {
        console.error('Failed to update sampling settings', err);
    }
}

async function onSettingsChanged() {
    await saveSettingsAndRestart();
}

// ── Chart data ─────────────────────────────────────────────────────
const chartTags = computed<ChartTag[]>(() =>
    state.settings.tags.map((t) => ({
        id: t.id,
        name: t.name,
        color: t.color,
        dataType: t.dataType,
        yAxis: t.yAxis,
        enabled: t.enabled,
    })),
);
const chartAxes = computed<ChartAxis[]>(() =>
    state.settings.yAxes.map((a) => ({ ...a })),
);

// ── Tag value display helpers ──────────────────────────────────────
const liveValues = reactive<Record<string, string>>({});

function tagValue(id: string) {
    return liveValues[id] ?? '-';
}
function tagMin(id: string) {
    const r = state.sampledRange[id];
    return r ? formatNumber(r.min) : '-';
}
function tagMax(id: string) {
    const r = state.sampledRange[id];
    return r ? formatNumber(r.max) : '-';
}

// ── Tag DataGrid state & helpers ───────────────────────────────────
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
    if (!state.settings.tags.length) return false;
    return state.settings.tags.every((t) => t.enabled);
});

function toggleAllTags(e: Event) {
    const checked = (e.target as HTMLInputElement).checked;
    state.settings.tags.forEach((t) => {
        t.enabled = checked;
    });
    onSettingsChanged();
}

const sortedTags = computed(() => {
    let list = state.settings.tags;
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
            const vA = parseFloat(liveValues[a.id] ?? '');
            const vB = parseFloat(liveValues[b.id] ?? '');
            if (Number.isFinite(vA) && Number.isFinite(vB))
                return (vA - vB) * dir;
            return (
                (liveValues[a.id] || '').localeCompare(liveValues[b.id] || '') *
                dir
            );
        }
        if (k === 'min') {
            const mA =
                state.sampledRange[a.id]?.min ??
                (sortAsc.value ? Infinity : -Infinity);
            const mB =
                state.sampledRange[b.id]?.min ??
                (sortAsc.value ? Infinity : -Infinity);
            return (mA - mB) * dir;
        }
        if (k === 'max') {
            const mA =
                state.sampledRange[a.id]?.max ??
                (sortAsc.value ? -Infinity : Infinity);
            const mB =
                state.sampledRange[b.id]?.max ??
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

// ── Dialog save handlers ───────────────────────────────────────────
async function onPlcTagsSave() {
    await onSettingsChanged();
}
async function onYAxesSave() {
    await onSettingsChanged();
}

// ── Menu actions ───────────────────────────────────────────────────
async function menuLoadSettings() {
    try {
        const loaded = await backend()?.LoadPlcTagSettings(
            t('menu.load_settings'),
        );
        if (loaded?.plcLinks?.length) {
            state.settings.plcLinks = loaded.plcLinks;
            state.settings.tags = loaded.tags ?? [];
            await saveSettingsAndRestart();
            state.statusMessage = t('status.loaded_settings');
        }
    } catch (err) {
        state.statusMessage = t('status.error_loading_settings', [err]);
    }
}

async function menuSaveSettings() {
    try {
        await backend()?.SavePlcTagSettings(
            state.settings.plcLinks,
            state.settings.tags,
            t('menu.save_settings'),
        );
        state.statusMessage = t('status.saved_settings');
    } catch (err) {
        state.statusMessage = t('status.error_saving_settings', [err]);
    }
}

async function menuTimer() {
    const value = await showPrompt(
        t('menu.set_timer_interval'),
        t('prompt.ms_range'),
        String(state.settings.pollIntervalMs),
    );
    if (value === null) return;
    const interval = Number(value);
    if (!Number.isInteger(interval) || interval < 10 || interval > 60000) {
        showMessage(
            t('menu.set_timer_interval'),
            t('prompt.invalid_interval', ['10', '60000']),
        );
        return;
    }
    state.settings.pollIntervalMs = interval;
    await saveSettingsAndRestart();
    state.statusMessage = t('status.interval_set', [interval]);
}

async function menuTrendWindow() {
    const value = await showPrompt(
        t('menu.x_settings'),
        t('prompt.s_range'),
        String(state.settings.timeWindowSeconds),
    );
    if (value === null) return;
    const seconds = Number(value);
    if (!Number.isInteger(seconds) || seconds < 30 || seconds > 86400) {
        showMessage(
            t('menu.x_settings'),
            t('prompt.invalid_window', ['30', '86400']),
        );
        return;
    }
    state.settings.timeWindowSeconds = seconds;
    await saveSettingsAndRestart();
    state.statusMessage = t('status.window_set', [seconds]);
}

async function menuInterpolation() {
    const value = await showPrompt(
        t('menu.samples_interpolation'),
        t('prompt.mode'),
        state.settings.interpolation,
        ['Line', 'Differential'],
    );
    if (value === null) return;
    state.settings.interpolation = value;
    await saveSettingsAndRestart();
    state.statusMessage = t('status.interpolation_set', [value]);
}

async function menuClear() {
    if (
        !(await showConfirm(t('menu.clear_records'), t('prompt.clear_confirm')))
    )
        return;
    chartRef.value?.clear();
    resetStats();
    Object.keys(liveValues).forEach((k) => delete liveValues[k]);
    state.statusMessage = t('status.records_cleared');
}

async function menuExport() {
    try {
        await backend()?.ExportCSV(t('menu.export_csv'));
        state.statusMessage = t('status.exported_data');
    } catch (err) {
        state.statusMessage = t('status.error_exporting', [err]);
    }
}

// ── Toolbar actions ────────────────────────────────────────────────
async function startSampling() {
    try {
        const api = backend();
        resetActualSampling();
        if (api?.StartPolling) await api.StartPolling(state.settings);
        state.isSampling = true;
        state.isPaused = false;
        chartRef.value?.setPaused(false);
        state.statusMessage = t('status.polling_started');
    } catch (err) {
        state.isSampling = false;
        resetActualSampling();
        state.statusMessage = t('status.polling_failed', [err]);
    }
}

function pauseSampling() {
    if (!state.isSampling) return;
    state.isPaused = !state.isPaused;
    chartRef.value?.setPaused(state.isPaused);
    state.statusMessage = state.isPaused
        ? t('status.chart_paused_polling')
        : t('status.chart_resumed');
}

async function stopSampling() {
    try {
        await backend()?.StopPolling();
    } catch {
        /* reset locally */
    }
    state.isSampling = false;
    state.isPaused = false;
    chartRef.value?.setPaused(false);
    resetActualSampling();
    state.statusMessage = t('status.polling_stopped');
}

// ── Chart drag start ───────────────────────────────────────────────
function onChartDragStart() {
    if (!state.isSampling || state.isPaused) return;
    state.isPaused = true;
    chartRef.value?.setPaused(true);
    state.statusMessage = 'Chart paused for review; polling continues';
}

function onPromptClose(result: string | null) {
    promptOpen.value = false;
    promptResolve?.(result);
    promptResolve = null;
}

function onMsgClose(result: boolean) {
    uiState.msgOpen = false;
    uiState.msgResolve?.(result);
    uiState.msgResolve = null;
}

// ── Resizer ────────────────────────────────────────────────────────
function onResizerPointerDown(e: PointerEvent) {
    const tagsArea = document.getElementById('tagsArea');
    if (!tagsArea) return;
    const startY = e.clientY;
    const startH = tagsArea.offsetHeight;

    function onMove(ev: PointerEvent) {
        const delta = startY - ev.clientY;
        tagsArea!.style.height = `${Math.max(50, startH + delta)}px`;
    }
    function onUp() {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
    }
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
}

// ── Lifecycle ──────────────────────────────────────────────────────
onMounted(async () => {
    try {
        const lang = await backend()?.GetSystemLanguage?.();
        if (lang && lang.toLowerCase().startsWith('zh')) {
            locale.value = 'zh';
        } else if (lang) {
            locale.value = 'en';
        }
    } catch (e) {
        console.error('Error fetching OS language', e);
    }

    // Wails event listener
    const rt = wailsRuntime();
    if (rt?.EventsOn) {
        rt.EventsOn('poll_update', (update: any) => {
            if (!state.isSampling) return;
            const tag = getTag(update.tagId);
            const valStr = update.value || '-';
            const numVal = Number(update.numericValue);

            liveValues[update.tagId] = valStr;
            updateTagValue(update.tagId, valStr, numVal);
            markPlcForTag(update.tagId, update.quality);

            if (
                update.numericValue !== null &&
                update.numericValue !== undefined
            ) {
                chartRef.value?.addDataPoint(
                    update.tagId,
                    update.timestamp,
                    numVal,
                );
            }
        });

        rt.EventsOn('poll_timing', (timing: any) => {
            if (!state.isSampling) return;
            const ms = Number(timing?.actualIntervalMs);
            const link = String(timing?.plcLink || '');
            if (!link || !Number.isFinite(ms) || ms <= 0) return;
            const next = new Map(actualByPlc.value);
            next.set(link, Math.round(ms));
            actualByPlc.value = next;
        });
    }

    // Browser preview fallback (dev mode only)
    if (import.meta.env.DEV) {
        let lastFallback = 0;
        fallbackTimer = window.setInterval(() => {
            if (rt || !state.isSampling) return;
            const nowMs = Date.now();
            if (lastFallback > 0) {
                const next = new Map(actualByPlc.value);
                next.set('preview', nowMs - lastFallback);
                actualByPlc.value = next;
            }
            lastFallback = nowMs;
            const now = new Date().toISOString();
            state.settings.tags
                .filter((t) => t.enabled)
                .forEach((tag) => {
                    const value = Math.random() * 100;
                    chartRef.value?.addDataPoint(tag.id, now, value);
                    liveValues[tag.id] = value.toFixed(2);
                    updateTagValue(tag.id, value.toFixed(2), value);
                });
        }, 100);
    }

    if (typeof window !== 'undefined') {
        window.addEventListener('keydown', handleKeydown);
    }
});

onBeforeUnmount(() => {
    if (typeof window !== 'undefined') {
        window.removeEventListener('keydown', handleKeydown);
    }
    if (fallbackTimer !== null) clearInterval(fallbackTimer);
});
</script>
