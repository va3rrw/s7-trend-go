<template>
    <footer class="statusbar">
        <div class="status-item">{{ statusMessage }}</div>
        <div class="status-sep" />
        <div class="status-item">
            {{ $t('status.window') }}
            {{ timeWindowSeconds }} s
        </div>
        <div class="status-sep" />
        <div class="status-item">
            {{ $t('status.interval') }}
            {{ pollIntervalMs }} ms
        </div>
        <div class="status-sep" />
        <div class="status-item">
            {{ $t('status.actual') }}
            {{ actualSamplingText }}
        </div>
    </footer>
</template>

<script setup lang="ts">
defineProps<{
    statusMessage: string;
    timeWindowSeconds: number;
    pollIntervalMs: number;
    actualSamplingText: string;
}>();
</script>
