import { state, backend } from '../store';

export interface SamplingChartRef {
    setPaused(paused: boolean): void;
    addDataPoint(id: string, timestamp: string, value: number): void;
    clear(): void;
}

/**
 * Save settings and restart live polling if sampling is currently active
 */
export async function saveSettingsAndRestart(): Promise<void> {
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

/**
 * Trigger settings change propagation
 */
export async function onSettingsChanged(): Promise<void> {
    await saveSettingsAndRestart();
}

/**
 * Start PLC sampling
 */
export async function startSampling(
    t: (key: string, args?: any[]) => string,
    chartRef?: SamplingChartRef | null,
    onResetActual?: () => void,
): Promise<void> {
    try {
        const api = backend();
        onResetActual?.();
        if (api?.StartPolling) {
            await api.StartPolling(state.settings);
        }
        state.isSampling = true;
        state.isPaused = false;
        chartRef?.setPaused(false);
        state.statusMessage = t('status.polling_started');
    } catch (err) {
        state.isSampling = false;
        onResetActual?.();
        state.statusMessage = t('status.polling_failed', [err]);
    }
}

/**
 * Pause or resume sampling / chart review
 */
export function pauseSampling(
    t: (key: string, args?: any[]) => string,
    chartRef?: SamplingChartRef | null,
): void {
    if (!state.isSampling) return;
    state.isPaused = !state.isPaused;
    chartRef?.setPaused(state.isPaused);
    state.statusMessage = state.isPaused
        ? t('status.chart_paused_polling')
        : t('status.chart_resumed');
}

/**
 * Stop sampling
 */
export async function stopSampling(
    t: (key: string, args?: any[]) => string,
    chartRef?: SamplingChartRef | null,
    onResetActual?: () => void,
): Promise<void> {
    try {
        await backend()?.StopPolling();
    } catch {
        /* reset locally */
    }
    state.isSampling = false;
    state.isPaused = false;
    chartRef?.setPaused(false);
    onResetActual?.();
    state.statusMessage = t('status.polling_stopped');
}
