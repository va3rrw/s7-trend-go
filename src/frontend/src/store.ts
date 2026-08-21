import { reactive } from 'vue';
import type { AppSettings, TagSettings } from './types';
import { defaultSettings } from './types';

/** Reactive application state shared across all components */
export const state = reactive({
    settings: defaultSettings() as AppSettings,
    isSampling: false,
    isPaused: false,
    statusMessage: 'Ready',
    sampledRange: {} as Record<string, { min: number; max: number }>,
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
export interface BackendApi {
    CheckStatus(plcLinkName: string): Promise<boolean>;
    ClearHistory(): Promise<void>;
    Connect(
        plcLinkName: string,
        ipAddress: string,
        rack: number,
        slot: number,
    ): Promise<void>;
    Disconnect(plcLinkName: string): Promise<void>;
    DisconnectAll(): Promise<void>;
    ExportCSV(title: string): Promise<void>;
    GetHistoryRange(
        tagIds: string[],
        startMs: number,
        endMs: number,
    ): Promise<Record<string, Array<{ t: number; v: number }>>>;
    GetLastSettingsPath(): Promise<string>;
    GetSettings(): Promise<AppSettings>;
    GetSystemLanguage(): Promise<string>;
    GetVersion(): Promise<string>;
    HasSettingsChanged(): Promise<boolean>;
    LoadSettingsFile(title: string): Promise<AppSettings>;
    QuitApp(): Promise<void>;
    RecordSample(
        tagId: string,
        timestamp: number,
        value: number,
    ): Promise<void>;
    SaveCurrentSettings(): Promise<void>;
    SaveSettings(settings: AppSettings): Promise<void>;
    SaveSettingsFile(settings: AppSettings, title: string): Promise<void>;
    StartPolling(settings: AppSettings): Promise<void>;
    StopPolling(): Promise<void>;
    TestConnection(link: {
        name: string;
        ipAddress: string;
        rack: number;
        slot: number;
        isConnected: boolean;
    }): Promise<void>;
}

export interface WailsRuntimeApi {
    EventsOn(eventName: string, callback: (...data: any) => void): () => void;
    EventsOff(eventName: string, ...additionalEventNames: string[]): void;
    EventsOnce(eventName: string, callback: (...data: any) => void): void;
    EventsEmit(eventName: string, ...optionalData: any): void;
    WindowSetTitle(title: string): void;
    Quit(): void;
}

/** Backend API accessor – returns typed Wails Go bindings or undefined in browser preview / tests */
export function backend(): BackendApi | undefined {
    if (typeof window === 'undefined') return undefined;
    return (window as any).go?.backend?.App;
}

/** Wails runtime accessor */
export function wailsRuntime(): WailsRuntimeApi | undefined {
    if (typeof window === 'undefined') return undefined;
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
        const current = state.sampledRange[id] ?? {
            min: numericValue,
            max: numericValue,
        };
        current.min = Math.min(current.min, numericValue);
        current.max = Math.max(current.max, numericValue);
        state.sampledRange[id] = current;
    }
}

export function resetStats(tagId?: string) {
    if (tagId) {
        delete state.sampledRange[tagId];
    } else {
        state.sampledRange = {};
    }
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
