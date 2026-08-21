<template>
    <div id="app">
        <!-- Menu Bar -->
        <AppMenuBar
            :cursors-enabled="cursorsEnabled"
            @load-settings="menuLoadSettings(t)"
            @save-settings="menuSaveSettings(t)"
            @open-plc-tags="plcTagsOpen = true"
            @open-trend-window="menuTrendWindow(t, showPrompt, showMessage)"
            @open-y-axes="yAxesOpen = true"
            @toggle-cursors="toggleCursors"
            @export-csv="menuExport(t)"
            @set-timer="menuTimer(t, showPrompt, showMessage)"
            @start-sampling="startSampling(t, chartRef, resetActualSampling)"
            @pause-sampling="pauseSampling(t, chartRef)"
            @stop-sampling="stopSampling(t, chartRef, resetActualSampling)"
            @set-interpolation="menuInterpolation(t, showPrompt)"
            @clear-records="menuClear(t, showConfirm, chartRef, liveValues)"
            @set-locale="locale = $event"
            @open-about="aboutOpen = true" />

        <!-- Toolbar -->
        <AppToolbar
            :cursors-enabled="cursorsEnabled"
            @open-plc-tags="plcTagsOpen = true"
            @open-y-axes="yAxesOpen = true"
            @open-trend-window="menuTrendWindow(t, showPrompt, showMessage)"
            @toggle-cursors="toggleCursors"
            @start-sampling="startSampling(t, chartRef, resetActualSampling)"
            @pause-sampling="pauseSampling(t, chartRef)"
            @stop-sampling="stopSampling(t, chartRef, resetActualSampling)" />

        <!-- Main Content Area -->
        <div class="main-content">
            <TrendChartView
                ref="chartRef"
                :tags="chartTags"
                :axes="chartAxes"
                :time-window-seconds="state.settings.timeWindowSeconds"
                :interpolation="state.settings.interpolation"
                :cursors-enabled="cursorsEnabled"
                :is-sampling="state.isSampling"
                :is-paused="state.isPaused"
                @drag-start="onChartDragStart"
                @window-change="onChartWindowChange"
                @cursor-change="onCursorChange"
                @resume="pauseSampling(t, chartRef)" />

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
            <TagsDataGrid
                :tags="state.settings.tags"
                :live-values="liveValues"
                :sampled-range="state.sampledRange"
                @edit-tag="onRowDblClick"
                @settings-change="onSettingsChanged" />
        </div>

        <!-- Status Bar -->
        <AppStatusBar
            :status-message="state.statusMessage"
            :time-window-seconds="state.settings.timeWindowSeconds"
            :poll-interval-ms="state.settings.pollIntervalMs"
            :actual-sampling-text="actualSamplingText" />
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

    <!-- App Exit Confirmation Dialog -->
    <ExitConfirmDialog
        :open="exitConfirmOpen"
        @save="onExitSave"
        @dont-save="onExitDontSave"
        @cancel="onExitCancel" />
</template>

<script setup lang="ts">
import { ref, reactive, shallowRef, computed, onMounted, onBeforeUnmount } from 'vue';
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
    uiState,
    showMessage,
} from './store';

// Subcomponents
import AppMenuBar from './components/AppMenuBar.vue';
import AppToolbar from './components/AppToolbar.vue';
import AppStatusBar from './components/AppStatusBar.vue';
import TagsDataGrid from './components/TagsDataGrid.vue';
import TrendChartView from './components/TrendChartView.vue';
import MeasurementStrip from './components/MeasurementStrip.vue';
import PlcTagsDialog from './components/PlcTagsDialog.vue';
import YAxesDialog from './components/YAxesDialog.vue';
import PromptDialog from './components/PromptDialog.vue';
import MessageDialog from './components/MessageDialog.vue';
import AboutDialog from './components/AboutDialog.vue';
import TagEditorDialog from './components/TagEditorDialog.vue';
import ExitConfirmDialog from './components/ExitConfirmDialog.vue';

// Utils
import {
    startSampling,
    pauseSampling,
    stopSampling,
    onSettingsChanged,
} from './utils/sampling';
import {
    menuLoadSettings,
    menuSaveSettings,
    menuTimer,
    menuTrendWindow,
    menuInterpolation,
    menuClear,
    menuExport,
} from './utils/menuActions';
import { handleKeydown } from './utils/shortcuts';

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

// ── Dialog visibility & helpers ────────────────────────────────────
const plcTagsOpen = ref(false);
const yAxesOpen = ref(false);
const aboutOpen = ref(false);

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

// ── Double-click tag editor ────────────────────────────────────────
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

// ── App Exit Confirmation ──────────────────────────────────────────
const exitConfirmOpen = ref(false);

async function onExitSave() {
    exitConfirmOpen.value = false;
    try {
        const api = backend();
        if (api?.SaveCurrentSettings) {
            await api.SaveCurrentSettings();
        } else if (api?.SaveSettingsFile) {
            await api.SaveSettingsFile(state.settings, t('menu.save_settings'));
        }
        if (api?.QuitApp) {
            await api.QuitApp();
        } else {
            wailsRuntime()?.Quit?.();
        }
    } catch {
        // User cancelled file save dialog; keep app open
    }
}

async function onExitDontSave() {
    exitConfirmOpen.value = false;
    const api = backend();
    if (api?.QuitApp) {
        await api.QuitApp();
    } else {
        wailsRuntime()?.Quit?.();
    }
}

function onExitCancel() {
    exitConfirmOpen.value = false;
}

// ── Polling Interval Tracking ──────────────────────────────────────
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

// ── Live Values ────────────────────────────────────────────────────
const liveValues = shallowRef<Record<string, string>>({});
let pendingLiveValues: Record<string, string> = {};
let liveValuesRaf: number | null = null;

function setLiveValue(tagId: string, valStr: string) {
    pendingLiveValues[tagId] = valStr;
    if (liveValuesRaf === null && typeof window !== 'undefined') {
        liveValuesRaf = window.requestAnimationFrame(() => {
            liveValuesRaf = null;
            liveValues.value = { ...liveValues.value, ...pendingLiveValues };
            pendingLiveValues = {};
        });
    }
}

// ── Dialog save handlers ───────────────────────────────────────────
async function onPlcTagsSave() {
    await onSettingsChanged();
}
async function onYAxesSave() {
    await onSettingsChanged();
}

// ── Chart drag start ───────────────────────────────────────────────
function onChartDragStart() {
    if (!state.isSampling || state.isPaused) return;
    state.isPaused = true;
    chartRef.value?.setPaused(true);
    state.statusMessage = 'Chart paused for review; polling continues';
}

function onChartWindowChange(seconds: number) {
    state.settings.timeWindowSeconds = seconds;
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

// ── Keyboard Shortcuts ─────────────────────────────────────────────
function onKeydown(e: KeyboardEvent) {
    handleKeydown(e, {
        hasModalOpen: () =>
            plcTagsOpen.value ||
            yAxesOpen.value ||
            aboutOpen.value ||
            promptOpen.value ||
            uiState.msgOpen ||
            dblEditOpen.value,
        toggleCursors,
        menuSaveSettings: () => menuSaveSettings(t),
        menuLoadSettings: () => menuLoadSettings(t),
        menuTrendWindow: () => menuTrendWindow(t, showPrompt, showMessage),
        openYAxes: () => {
            yAxesOpen.value = true;
        },
        openPlcTags: () => {
            plcTagsOpen.value = true;
        },
        menuExport: () => menuExport(t),
        startSampling: () => startSampling(t, chartRef.value, resetActualSampling),
        pauseSampling: () => pauseSampling(t, chartRef.value),
        stopSampling: () => stopSampling(t, chartRef.value, resetActualSampling),
    });
}

// ── Lifecycle ──────────────────────────────────────────────────────
let fallbackTimer: number | null = null;
let unlistenPollUpdate: (() => void) | null = null;
let unlistenPollTiming: (() => void) | null = null;
let unlistenRequestExit: (() => void) | null = null;

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

    try {
        const initialSettings = await backend()?.GetSettings?.();
        if (initialSettings && initialSettings.plcLinks?.length) {
            state.settings = initialSettings;
        }
    } catch (e) {
        console.error('Error fetching initial settings', e);
    }

    // Wails event listeners
    const rt = wailsRuntime();
    if (rt?.EventsOn) {
        unlistenRequestExit = rt.EventsOn('app:request-exit', () => {
            exitConfirmOpen.value = true;
        });

        unlistenPollUpdate = rt.EventsOn('poll_update', (update: any) => {
            if (!state.isSampling) return;
            const valStr = update.value || '-';
            const numVal = Number(update.numericValue);

            setLiveValue(update.tagId, valStr);
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

        unlistenPollTiming = rt.EventsOn('poll_timing', (timing: any) => {
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
                    setLiveValue(tag.id, value.toFixed(2));
                    updateTagValue(tag.id, value.toFixed(2), value);
                });
        }, 100);
    }

    if (typeof window !== 'undefined') {
        window.addEventListener('keydown', onKeydown);
    }
});

onBeforeUnmount(() => {
    if (typeof window !== 'undefined') {
        window.removeEventListener('keydown', onKeydown);
    }
    if (liveValuesRaf !== null && typeof window !== 'undefined') {
        window.cancelAnimationFrame(liveValuesRaf);
        liveValuesRaf = null;
    }
    if (fallbackTimer !== null) {
        clearInterval(fallbackTimer);
        fallbackTimer = null;
    }
    if (typeof unlistenRequestExit === 'function') {
        unlistenRequestExit();
        unlistenRequestExit = null;
    } else {
        const rt = wailsRuntime();
        rt?.EventsOff?.('app:request-exit');
    }
    if (typeof unlistenPollUpdate === 'function') {
        unlistenPollUpdate();
        unlistenPollUpdate = null;
    } else {
        const rt = wailsRuntime();
        rt?.EventsOff?.('poll_update');
    }
    if (typeof unlistenPollTiming === 'function') {
        unlistenPollTiming();
        unlistenPollTiming = null;
    } else {
        const rt = wailsRuntime();
        rt?.EventsOff?.('poll_timing');
    }
});
</script>
