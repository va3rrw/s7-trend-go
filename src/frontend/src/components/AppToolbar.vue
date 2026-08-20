<template>
    <div class="toolbar">
        <button
            class="tool-btn"
            :title="$t('toolbar.plc_tag_settings')"
            @click="emit('openPlcTags')">
            <span
                class="i-ix-plc-tag icon-glyph"
                aria-hidden="true" />
        </button>
        <div class="separator" />
        <button
            class="tool-btn"
            :title="$t('toolbar.y_axis_settings')"
            @click="emit('openYAxes')">
            <span
                class="i-ix-y-axis-settings icon-glyph"
                aria-hidden="true" />
        </button>
        <button
            class="tool-btn"
            :title="$t('toolbar.x_axis_settings')"
            @click="emit('openTrendWindow')">
            <span
                class="i-ix-x-axis-settings icon-glyph"
                aria-hidden="true" />
        </button>
        <div class="separator" />
        <button
            class="tool-btn"
            :class="{ active: cursorsEnabled }"
            :title="$t('toolbar.cursors')"
            @click="emit('toggleCursors')">
            <span
                class="i-ix-ruler-horizontal icon-glyph"
                aria-hidden="true" />
        </button>
        <div class="separator" />
        <button
            class="tool-btn start"
            :title="$t('toolbar.start_sampling')"
            :disabled="state.isSampling"
            @click="emit('startSampling')">
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
            @click="emit('pauseSampling')">
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
            @click="emit('stopSampling')">
            <span
                class="i-ix-stop icon-glyph"
                aria-hidden="true" />
        </button>
    </div>
</template>

<script setup lang="ts">
import { state } from '../store';

defineProps<{
    cursorsEnabled: boolean;
}>();

const emit = defineEmits<{
    openPlcTags: [];
    openYAxes: [];
    openTrendWindow: [];
    toggleCursors: [];
    startSampling: [];
    pauseSampling: [];
    stopSampling: [];
}>();
</script>
