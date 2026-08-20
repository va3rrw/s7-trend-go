<template>
    <nav class="menubar">
        <div class="menu-item">
            <span>{{ $t('menu.file') }}</span>
            <div class="dropdown">
                <a
                    href="#"
                    @click.prevent="emit('loadSettings')">
                    <span>{{ $t('menu.load_settings') }}</span>
                    <span class="menu-shortcut">Ctrl+L</span>
                </a>
                <a
                    href="#"
                    @click.prevent="emit('saveSettings')">
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
                    @click.prevent="emit('openPlcTags')">
                    <span>{{ $t('menu.plc_tag_settings') }}</span>
                    <span class="menu-shortcut">Ctrl+T</span>
                </a>
                <a
                    href="#"
                    @click.prevent="emit('openTrendWindow')">
                    <span>{{ $t('menu.x_settings') }}</span>
                    <span class="menu-shortcut">Ctrl+X</span>
                </a>
                <a
                    href="#"
                    @click.prevent="emit('openYAxes')">
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
                    @click.prevent="emit('toggleCursors')">
                    <span
                        >{{ cursorsEnabled ? '✓ ' : ''
                        }}{{ $t('menu.measurement_cursors') }}</span
                    >
                    <span class="menu-shortcut">Ctrl+M</span>
                </a>
                <hr />
                <a
                    href="#"
                    @click.prevent="emit('exportCsv')">
                    <span>{{ $t('menu.export_csv') }}</span>
                </a>
            </div>
        </div>
        <div class="menu-item">
            <span>{{ $t('menu.trace') }}</span>
            <div class="dropdown">
                <a
                    href="#"
                    @click.prevent="emit('setTimer')">
                    <span>{{ $t('menu.set_timer_interval') }}</span>
                </a>
                <hr />
                <a
                    href="#"
                    :class="{ disabled: state.isSampling }"
                    @click.prevent="!state.isSampling && emit('startSampling')">
                    <span>{{ $t('menu.start') }}</span>
                    <span class="menu-shortcut">F5</span>
                </a>
                <a
                    href="#"
                    :class="{ disabled: !state.isSampling }"
                    @click.prevent="state.isSampling && emit('pauseSampling')">
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
                    @click.prevent="state.isSampling && emit('stopSampling')">
                    <span>{{ $t('menu.stop') }}</span>
                    <span class="menu-shortcut">Shift+F5</span>
                </a>
                <hr />
                <a
                    href="#"
                    @click.prevent="emit('setInterpolation')">
                    <span>{{ $t('menu.samples_interpolation') }}</span>
                </a>
                <hr />
                <a
                    href="#"
                    @click.prevent="emit('clearRecords')">
                    <span>{{ $t('menu.clear_records') }}</span>
                </a>
            </div>
        </div>
        <div class="menu-item">
            <span>{{ $t('menu.help') }}</span>
            <div class="dropdown">
                <a
                    href="#"
                    @click.prevent="emit('setLocale', 'en')">
                    <span>English</span>
                </a>
                <a
                    href="#"
                    @click.prevent="emit('setLocale', 'zh')">
                    <span>简体中文</span>
                </a>
                <hr />
                <a
                    href="#"
                    @click.prevent="emit('openAbout')">
                    <span>{{ $t('menu.about') }}</span>
                </a>
            </div>
        </div>
    </nav>
</template>

<script setup lang="ts">
import { state } from '../store';

defineProps<{
    cursorsEnabled: boolean;
}>();

const emit = defineEmits<{
    loadSettings: [];
    saveSettings: [];
    openPlcTags: [];
    openTrendWindow: [];
    openYAxes: [];
    toggleCursors: [];
    exportCsv: [];
    setTimer: [];
    startSampling: [];
    pauseSampling: [];
    stopSampling: [];
    setInterpolation: [];
    clearRecords: [];
    setLocale: [lang: string];
    openAbout: [];
}>();
</script>
