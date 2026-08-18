<template>
    <div id="app">
        <!-- Menu Bar — layout mirrors old S7TrendValues (File / Edit / Tools / Trace / Help) -->
        <nav class="menubar">
            <div class="menu-item">
                <span>{{ $t('menu.file') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="menuLoadPlc"
                        >{{ $t('menu.load_plc_settings') }}</a
                    >
                    <a
                        href="#"
                        @click.prevent="menuSavePlc"
                        >{{ $t('menu.save_plc_settings') }}</a
                    >
                    <hr />
                    <a
                        href="#"
                        @click.prevent="menuLoadTags"
                        >{{ $t('menu.load_channel_settings') }}</a
                    >
                    <a
                        href="#"
                        @click.prevent="menuSaveTags"
                        >{{ $t('menu.save_channel_settings') }}</a
                    >
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.edit') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="plcOpen = true"
                        >{{ $t('menu.plc_settings') }}</a
                    >
                    <a
                        href="#"
                        @click.prevent="tagsOpen = true"
                        >{{ $t('menu.channel_settings') }}</a
                    >
                    <a
                        href="#"
                        @click.prevent="menuTrendWindow"
                        >{{ $t('menu.x_settings') }}</a
                    >
                    <a
                        href="#"
                        @click.prevent="yAxesOpen = true"
                        >{{ $t('menu.y_settings') }}</a
                    >
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.tools') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="menuExport"
                        >{{ $t('menu.export_csv') }}</a
                    >
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.trace') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="menuTimer"
                        >{{ $t('menu.set_timer_interval') }}</a
                    >
                    <hr />
                    <a
                        href="#"
                        :class="{ disabled: state.isSampling }"
                        @click.prevent="!state.isSampling && startSampling()"
                        >{{ $t('menu.start') }}</a
                    >
                    <a
                        href="#"
                        :class="{ disabled: !state.isSampling }"
                        @click.prevent="state.isSampling && pauseSampling()"
                        >{{
                            state.isPaused
                                ? $t('menu.resume')
                                : $t('menu.pause')
                        }}</a
                    >
                    <a
                        href="#"
                        :class="{ disabled: !state.isSampling }"
                        @click.prevent="state.isSampling && stopSampling()"
                        >{{ $t('menu.stop') }}</a
                    >
                    <hr />
                    <a
                        href="#"
                        @click.prevent="menuInterpolation"
                        >{{ $t('menu.samples_interpolation') }}</a
                    >
                    <hr />
                    <a
                        href="#"
                        @click.prevent="menuClear"
                        >{{ $t('menu.clear_records') }}</a
                    >
                </div>
            </div>
            <div class="menu-item">
                <span>{{ $t('menu.help') }}</span>
                <div class="dropdown">
                    <a
                        href="#"
                        @click.prevent="locale = 'en'"
                        >English</a
                    >
                    <a
                        href="#"
                        @click.prevent="locale = 'zh'"
                        >简体中文</a
                    >
                    <hr />
                    <a
                        href="#"
                        @click.prevent="aboutOpen = true"
                        >{{ $t('menu.about') }}</a
                    >
                </div>
            </div>
        </nav>

        <!-- Toolbar -->
        <div class="toolbar">
            <button
                class="tool-btn"
                :title="$t('toolbar.plcs')"
                @click="plcOpen = true">
                <span
                    class="i-ix-plc-device icon-glyph"
                    aria-hidden="true" />
            </button>
            <button
                class="tool-btn"
                :title="$t('toolbar.tag_settings')"
                @click="tagsOpen = true">
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
                @drag-start="onChartDragStart" />

            <!-- Resizer -->
            <div
                class="resizer"
                @pointerdown="onResizerPointerDown" />

            <!-- Tags DataGrid Area -->
            <div
                id="tagsArea"
                class="tags-area">
                <div class="tags-grids">
                    <div class="tags-grid-container">
                        <table
                            class="datagrid"
                            id="gridLeft">
                            <thead>
                                <tr>
                                    <th>{{ $t('grid.on') }}</th>
                                    <th>{{ $t('grid.tag') }}</th>
                                    <th>{{ $t('grid.plc') }}</th>
                                    <th>{{ $t('grid.value') }}</th>
                                    <th>{{ $t('grid.min') }}</th>
                                    <th>{{ $t('grid.max') }}</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr
                                    v-for="tag in leftTags"
                                    :key="tag.id"
                                    :style="{
                                        backgroundColor:
                                            tag.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(tag)">
                                    <td>
                                        <input
                                            type="checkbox"
                                            :checked="tag.enabled"
                                            @change="
                                                tag.enabled = (
                                                    $event.target as HTMLInputElement
                                                ).checked;
                                                onSettingsChanged();
                                            " />
                                    </td>
                                    <td>{{ tag.name }}</td>
                                    <td>{{ tag.plcLink }}</td>
                                    <td>{{ tagValue(tag.id) }}</td>
                                    <td>{{ tagMin(tag.id) }}</td>
                                    <td>{{ tagMax(tag.id) }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="grid-splitter" />
                    <div class="tags-grid-container">
                        <table
                            class="datagrid"
                            id="gridRight">
                            <thead>
                                <tr>
                                    <th>{{ $t('grid.on') }}</th>
                                    <th>{{ $t('grid.tag') }}</th>
                                    <th>{{ $t('grid.plc') }}</th>
                                    <th>{{ $t('grid.value') }}</th>
                                    <th>{{ $t('grid.min') }}</th>
                                    <th>{{ $t('grid.max') }}</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr
                                    v-for="tag in rightTags"
                                    :key="tag.id"
                                    :style="{
                                        backgroundColor:
                                            tag.color || 'transparent',
                                    }"
                                    @dblclick="onRowDblClick(tag)">
                                    <td>
                                        <input
                                            type="checkbox"
                                            :checked="tag.enabled"
                                            @change="
                                                tag.enabled = (
                                                    $event.target as HTMLInputElement
                                                ).checked;
                                                onSettingsChanged();
                                            " />
                                    </td>
                                    <td>{{ tag.name }}</td>
                                    <td>{{ tag.plcLink }}</td>
                                    <td>{{ tagValue(tag.id) }}</td>
                                    <td>{{ tagMin(tag.id) }}</td>
                                    <td>{{ tagMax(tag.id) }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
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
    <PlcDialog
        :open="plcOpen"
        @close="plcOpen = false"
        @save="onPlcSave" />
    <TagsDialog
        :open="tagsOpen"
        @close="tagsOpen = false"
        @save="onTagsSave" />
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
        @save="onDblEditSave"
        @close="dblEditOpen = false" />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import type { ChartTag, ChartAxis } from './chart';
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
import PlcDialog from './components/PlcDialog.vue';
import TagsDialog from './components/TagsDialog.vue';
import YAxesDialog from './components/YAxesDialog.vue';
import PromptDialog from './components/PromptDialog.vue';
import MessageDialog from './components/MessageDialog.vue';
import AboutDialog from './components/AboutDialog.vue';
import TagEditorDialog from './components/TagEditorDialog.vue';

// ── Chart ref ──────────────────────────────────────────────────────
const { t, locale } = useI18n();

const chartRef = ref<InstanceType<typeof TrendChartView>>();

// ── Dialog visibility ──────────────────────────────────────────────
const plcOpen = ref(false);
const tagsOpen = ref(false);
const yAxesOpen = ref(false);
const aboutOpen = ref(false);

// Prompt dialog
const promptOpen = ref(false);
const promptTitle = ref('');
const promptText = ref('');
const promptDefault = ref('');
const promptOptions = ref<string[]>();
let promptResolve: ((val: string | null) => void) | null = null;

// Message/Confirm dialog logic moved to store.ts

// Double-click edit on main grid – reuse TagEditorDialog directly
const dblEditOpen = ref(false);
const dblEditTag = ref<TagSettings | null>(null);
const dblEditIndex = ref(-1);

// ── Fallback timer for browser preview ─────────────────────────────
let fallbackTimer: number | null = null;

// ── Computed chart data ────────────────────────────────────────────
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

// ── Helpers ────────────────────────────────────────────────────────
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

async function saveSettingsAndRestart() {
    const api = backend();
    if (api?.SaveSettings) await api.SaveSettings(state.settings);
    if (state.isSampling && api?.StartPolling) {
        resetActualSampling();
        await api.StartPolling(state.settings);
    }
}

async function onSettingsChanged() {
    await saveSettingsAndRestart();
}

// ── Tag value display helpers ──────────────────────────────────────
function tagValue(id: string) {
    // We'll track live values in a separate reactive map
    return liveValues.value.get(id) ?? '-';
}
function tagMin(id: string) {
    const r = state.sampledRange.get(id);
    return r ? formatNumber(r.min) : '-';
}
function tagMax(id: string) {
    const r = state.sampledRange.get(id);
    return r ? formatNumber(r.max) : '-';
}

const liveValues = ref(new Map<string, string>());

// Calculate visible rows dynamically based on the tags area height
const visibleRows = ref(4);

// Split tags into left (up to visibleRows) and right (rest)
const leftTags = computed(() =>
    state.settings.tags.slice(0, visibleRows.value),
);
const rightTags = computed(() =>
    state.settings.tags.slice(visibleRows.value),
);

// ── Menu actions ───────────────────────────────────────────────────
async function menuLoadPlc() {
    try {
        const links = await backend()?.LoadPlcLinks(t('menu.load_plc_settings'));
        if (links?.length) {
            state.settings.plcLinks = links;
            await saveSettingsAndRestart();
            state.statusMessage = t('status.loaded_plcs');
        }
    } catch (err) {
        state.statusMessage = t('status.error_loading_plcs', [err]);
    }
}

async function menuSavePlc() {
    try {
        await backend()?.SavePlcLinks(state.settings.plcLinks, t('menu.save_plc_settings'));
        state.statusMessage = t('status.saved_plcs');
    } catch (err) {
        state.statusMessage = t('status.error_saving_plcs', [err]);
    }
}

async function menuLoadTags() {
    try {
        const tags = await backend()?.LoadTags(t('menu.load_channel_settings'));
        if (tags) {
            state.settings.tags = tags;
            await saveSettingsAndRestart();
            state.statusMessage = t('status.loaded_tags');
        }
    } catch (err) {
        state.statusMessage = t('status.error_loading_tags', [err]);
    }
}

async function menuSaveTags() {
    try {
        await backend()?.SaveTags(state.settings.tags, t('menu.save_channel_settings'));
        state.statusMessage = t('status.saved_tags');
    } catch (err) {
        state.statusMessage = t('status.error_saving_tags', [err]);
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
    if (!Number.isInteger(interval) || interval < 10 || interval > 6000) {
        showMessage(t('menu.set_timer_interval'), t('prompt.invalid_interval', ['10', '6000']));
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
        showMessage(t('menu.x_settings'), t('prompt.invalid_window', ['30', '86400']));
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
        !(await showConfirm(
            t('menu.clear_records'),
            t('prompt.clear_confirm'),
        ))
    )
        return;
    state.recordsHistory.length = 0;
    chartRef.value?.clear();
    resetStats();
    liveValues.value.clear();
    state.statusMessage = t('status.records_cleared');
}

async function menuExport() {
    try {
        await backend()?.ExportCSV(state.recordsHistory, t('menu.export_csv'));
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

// ── Actual sampling interval (max across PLC links) ────────────────
/** Per-PLC measured cycle period in ms */
const actualByPlc = ref(new Map<string, number>());

const actualSamplingText = computed(() => {
    if (!state.isSampling) return '—';
    const values = [...actualByPlc.value.values()];
    if (values.length === 0) return '—';
    return `${Math.max(...values)} ms`;
});

function resetActualSampling() {
    actualByPlc.value = new Map();
}

// ── Chart drag start ───────────────────────────────────────────────
function onChartDragStart() {
    if (!state.isSampling || state.isPaused) return;
    state.isPaused = true;
    chartRef.value?.setPaused(true);
    state.statusMessage = 'Chart paused for review; polling continues';
}

// ── Double-click edit on main grid ─────────────────────────────────
function onRowDblClick(tag: TagSettings) {
    const index = state.settings.tags.findIndex((t) => t.id === tag.id);
    if (index < 0) return;
    dblEditTag.value = { ...tag };
    dblEditIndex.value = index;
    dblEditOpen.value = true;
}

function onDblEditSave(tag: TagSettings) {
    state.settings.tags[dblEditIndex.value] = tag;
    dblEditOpen.value = false;
    void onSettingsChanged();
}

// ── Dialog callbacks ───────────────────────────────────────────────
function onPlcSave() {
    void onSettingsChanged();
}
function onTagsSave() {
    void onSettingsChanged();
}
function onYAxesSave() {
    void onSettingsChanged();
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

let tagsAreaObserver: ResizeObserver | null = null;

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

    // Watch tagsArea height to dynamically size the left table
    const tagsArea = document.getElementById('tagsArea');
    if (tagsArea) {
        tagsAreaObserver = new ResizeObserver((entries) => {
            for (const entry of entries) {
                const height = entry.contentRect.height;
                // Border (2px) + Padding (4px) + Header (26px) = 32px overhead
                // Each row is ~23px high
                visibleRows.value = Math.max(1, Math.floor((height - 32) / 23));
            }
        });
        tagsAreaObserver.observe(tagsArea);
    }

    // Wails event listener
    const rt = wailsRuntime();
    if (rt?.EventsOn) {
        rt.EventsOn('poll_update', (update: any) => {
            if (!state.isSampling) return;
            const tag = getTag(update.tagId);
            const valStr = update.value || '-';
            const numVal = Number(update.numericValue);

            liveValues.value.set(update.tagId, valStr);
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
                if (tag) {
                    state.recordsHistory.push({
                        timestamp: update.timestamp,
                        tagName: tag.name,
                        address: tag.address,
                        value: numVal,
                    });
                    if (state.recordsHistory.length > 1000000) {
                        state.recordsHistory.splice(0, 100000);
                    }
                }
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
                    liveValues.value.set(tag.id, value.toFixed(2));
                    updateTagValue(tag.id, value.toFixed(2), value);
                    state.recordsHistory.push({
                        timestamp: now,
                        tagName: tag.name,
                        address: tag.address,
                        value,
                    });
                    if (state.recordsHistory.length > 1000000) {
                        state.recordsHistory.splice(0, 100000);
                    }
                });
        }, 100);
    }
});

onBeforeUnmount(() => {
    if (fallbackTimer !== null) clearInterval(fallbackTimer);
    tagsAreaObserver?.disconnect();
});
</script>
