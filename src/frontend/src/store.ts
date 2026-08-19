import { reactive } from 'vue';
import type { AppSettings, TagSettings } from './types';
import { defaultSettings } from './types';

/** Reactive application state shared across all components */
export const state = reactive({
    settings: defaultSettings() as AppSettings,
    isSampling: false,
    isPaused: false,
    statusMessage: 'Ready',
    sampledRange: new Map<string, { min: number; max: number }>(),
});

export const uiState = reactive({
    msgOpen: false,
    msgTitle: '',
    msgMessage: '',
    msgShowCancel: false,
    msgResolve: null as ((val: boolean) => void) | null,
});

export function showMessage(title: string, message: string, showCancel: boolean = false): Promise<boolean> {
    return new Promise((resolve) => {
        uiState.msgTitle = title;
        uiState.msgMessage = message;
        uiState.msgShowCancel = showCancel;
        uiState.msgOpen = true;
        uiState.msgResolve = resolve;
    });
}
/** Backend API accessor – returns the Wails Go bindings or undefined in browser preview */
export function backend(): any {
    return (window as any).go?.backend?.App;
}

/** Wails runtime accessor */
export function wailsRuntime(): any {
    return (window as any).runtime;
}

export function formatNumber(value: number): string {
    return Number.isInteger(value)
        ? String(value)
        : value.toFixed(3).replace(/0+$/, '').replace(/\.$/, '');
}

export function updateTagValue(
    id: string,
    value: string,
    numericValue?: number,
) {
    if (numericValue !== undefined && Number.isFinite(numericValue)) {
        const current = state.sampledRange.get(id) ?? {
            min: numericValue,
            max: numericValue,
        };
        current.min = Math.min(current.min, numericValue);
        current.max = Math.max(current.max, numericValue);
        state.sampledRange.set(id, current);
    }
}

export function resetStats() {
    state.sampledRange.clear();
}

export function getTag(id: string): TagSettings | undefined {
    return state.settings.tags.find((t) => t.id === id);
}

export function markPlcForTag(id: string, quality: string) {
    const tag = getTag(id);
    if (!tag) return;
    const link = state.settings.plcLinks.find((p) => p.name === tag.plcLink);
    if (link) link.isConnected = quality === 'Good';
}
