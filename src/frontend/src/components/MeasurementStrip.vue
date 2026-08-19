<template>
    <div
        v-if="open"
        class="measurement-panel">
        <!-- Top summary bar -->
        <div class="measurement-header">
            <div class="cursor-badges">
                <div class="cursor-badge cursor-a">
                    <span class="cursor-tag">A</span>
                    <span class="cursor-time">{{ formatTime(measurement?.cursorA) }}</span>
                </div>
                <div class="cursor-badge cursor-b">
                    <span class="cursor-tag">B</span>
                    <span class="cursor-time">{{ formatTime(measurement?.cursorB) }}</span>
                </div>
                <div class="delta-badge">
                    <span class="delta-label">ΔT:</span>
                    <span class="delta-val">{{ formatDeltaT(measurement?.deltaTSec, measurement?.deltaTMs) }}</span>
                </div>
            </div>

            <div class="measurement-actions">
                <button
                    class="btn btn-outline btn-xs"
                    :title="$t('measurement.reset_cursors')"
                    @click="emit('fitWindow')">
                    {{ $t('measurement.reset_cursors') }}
                </button>
                <button
                    class="btn-close-strip"
                    title="Close"
                    @click="emit('close')">
                    ✕
                </button>
            </div>
        </div>

        <!-- Measurement & Statistics Table -->
        <div class="measurement-table-wrapper">
            <table class="datagrid measurement-table">
                <thead>
                    <tr>
                        <th style="min-width: 140px">{{ $t('grid.tag') }}</th>
                        <th>{{ $t('measurement.cursor_a') }}</th>
                        <th>{{ $t('measurement.cursor_b') }}</th>
                        <th>{{ $t('measurement.delta_y') }}</th>
                        <th>{{ $t('measurement.slope') }}</th>
                        <th>{{ $t('grid.min') }}</th>
                        <th>{{ $t('grid.max') }}</th>
                        <th>{{ $t('measurement.mean') }}</th>
                        <th>{{ $t('measurement.std_dev') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr
                        v-for="tag in activeTags"
                        :key="tag.id">
                        <td class="tag-cell">
                            <span
                                class="tag-color-dot"
                                :style="{ backgroundColor: tag.color || '#93C5FD' }" />
                            <span class="tag-name">{{ tag.name }}</span>
                        </td>
                        <td class="mono-num">{{ formatVal(getStats(tag.id)?.valA) }}</td>
                        <td class="mono-num">{{ formatVal(getStats(tag.id)?.valB) }}</td>
                        <td
                            class="mono-num"
                            :class="deltaClass(getStats(tag.id)?.deltaY)">
                            {{ formatDeltaY(getStats(tag.id)?.deltaY) }}
                        </td>
                        <td class="mono-num">{{ formatSlope(getStats(tag.id)?.slope) }}</td>
                        <td class="mono-num">{{ formatVal(getStats(tag.id)?.min) }}</td>
                        <td class="mono-num">{{ formatVal(getStats(tag.id)?.max) }}</td>
                        <td class="mono-num">{{ formatVal(getStats(tag.id)?.mean) }}</td>
                        <td class="mono-num">{{ formatVal(getStats(tag.id)?.stdDev) }}</td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { TagSettings } from '../types';
import type { CursorMeasurement, TagStats } from '../measurement';

const props = defineProps<{
    open: boolean;
    measurement: CursorMeasurement | null;
    tags: TagSettings[];
}>();

const emit = defineEmits<{
    close: [];
    fitWindow: [];
}>();

const activeTags = computed(() => props.tags.filter((t) => t.enabled !== false));

function getStats(tagId: string): TagStats | undefined {
    return props.measurement?.tags[tagId];
}

function formatTime(ms: number | undefined | null): string {
    if (!ms || !Number.isFinite(ms)) return '--:--:--.---';
    const d = new Date(ms);
    const pad = (n: number, w = 2) => String(n).padStart(w, '0');
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(d.getMilliseconds(), 3)}`;
}

function formatDeltaT(sec: number | undefined | null, ms: number | undefined | null): string {
    if (sec === undefined || sec === null || ms === undefined || ms === null || !Number.isFinite(sec)) return '--';
    const sign = sec >= 0 ? '+' : '';
    return `${sign}${sec.toFixed(3)} s (${Math.abs(Math.round(ms))} ms)`;
}

function formatVal(v: number | null | undefined): string {
    if (v === null || v === undefined || !Number.isFinite(v)) return '-';
    return Number.isInteger(v) ? String(v) : v.toFixed(3).replace(/0+$/, '').replace(/\.$/, '');
}

function formatDeltaY(v: number | null | undefined): string {
    if (v === null || v === undefined || !Number.isFinite(v)) return '-';
    const sign = v > 0 ? '+' : '';
    return `${sign}${formatVal(v)}`;
}

function deltaClass(v: number | null | undefined): string {
    if (v === null || v === undefined || !Number.isFinite(v)) return '';
    if (v > 0) return 'delta-pos';
    if (v < 0) return 'delta-neg';
    return '';
}

function formatSlope(v: number | null | undefined): string {
    if (v === null || v === undefined || !Number.isFinite(v)) return '-';
    const sign = v > 0 ? '+' : '';
    return `${sign}${v.toFixed(3)} /s`;
}
</script>

<style scoped>
.measurement-panel {
    background: #0F172A;
    border: 1px solid #334155;
    border-radius: 4px;
    margin-bottom: 4px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    flex: 0 0 auto;
    max-height: 180px;
}

.measurement-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 8px;
    background: #1E293B;
    border-bottom: 1px solid #334155;
    font-size: 11px;
}

.cursor-badges {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
}

.cursor-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(0, 0, 0, 0.3);
    padding: 2px 8px;
    border-radius: 3px;
    border: 1px solid transparent;
}

.cursor-badge.cursor-a {
    border-color: #0284C7;
}
.cursor-badge.cursor-a .cursor-tag {
    background: #0284C7;
    color: white;
}

.cursor-badge.cursor-b {
    border-color: #D97706;
}
.cursor-badge.cursor-b .cursor-tag {
    background: #D97706;
    color: white;
}

.cursor-tag {
    font-weight: bold;
    font-size: 10px;
    padding: 1px 5px;
    border-radius: 2px;
    line-height: 1;
}

.cursor-time {
    font-family: ui-monospace, "Cascadia Mono", "Segoe UI Mono", Consolas, monospace;
    font-size: 11px;
    color: #F8FAFC;
}

.delta-badge {
    display: flex;
    align-items: center;
    gap: 4px;
    background: rgba(0, 0, 0, 0.3);
    padding: 2px 8px;
    border-radius: 3px;
    border: 1px solid #475569;
}

.delta-label {
    color: #94A3B8;
    font-weight: 600;
}

.delta-val {
    font-family: ui-monospace, "Cascadia Mono", "Segoe UI Mono", Consolas, monospace;
    color: #38BDF8;
    font-weight: bold;
}

.measurement-actions {
    display: flex;
    align-items: center;
    gap: 6px;
}

.btn-xs {
    padding: 2px 8px;
    font-size: 11px;
}

.btn-close-strip {
    background: transparent;
    border: none;
    color: #94A3B8;
    cursor: pointer;
    font-size: 12px;
    padding: 2px 6px;
    border-radius: 3px;
}
.btn-close-strip:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #F8FAFC;
}

.measurement-table-wrapper {
    max-height: 140px;
    overflow-y: auto;
    overflow-x: auto;
}

.measurement-table {
    width: 100%;
    margin: 0;
    border-collapse: collapse;
    font-size: 11px;
}

.measurement-table th {
    position: sticky;
    top: 0;
    background: #1E293B;
    color: #94A3B8;
    font-weight: 600;
    padding: 4px 8px;
    text-align: right;
    border-bottom: 1px solid #334155;
    white-space: nowrap;
}
.measurement-table th:first-child {
    text-align: left;
}

.measurement-table td {
    padding: 3px 8px;
    border-bottom: 1px solid #1E293B;
    text-align: right;
    color: #CBD5E1;
    white-space: nowrap;
}

.tag-cell {
    display: flex;
    align-items: center;
    gap: 6px;
    text-align: left;
}

.tag-color-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex: 0 0 8px;
}

.tag-name {
    font-weight: 500;
    color: #F8FAFC;
}

.mono-num {
    font-family: ui-monospace, "Cascadia Mono", "Segoe UI Mono", Consolas, monospace;
}

.delta-pos {
    color: #4ADE80;
}
.delta-neg {
    color: #F87171;
}
</style>
