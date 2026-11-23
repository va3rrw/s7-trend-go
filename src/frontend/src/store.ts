import { reactive } from 'vue';
import { Application, Events, Window } from '@wailsio/runtime';
import * as AppBinding from '../bindings/s7-trend-go/backend/app';
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
        maxPoints: number,
    ): Promise<Record<string, Array<{ t: number; v: number }>>>;
    GetLastSettingsPath(): Promise<string>;
    GetSettings(): Promise<AppSettings>;
    GetTreePanelWidth?(): Promise<number>;
    GetSystemLanguage(): Promise<string>;
    GetVersion(): Promise<string>;
    HasSettingsChanged(): Promise<boolean>;
    LoadSettingsFile(title: string): Promise<AppSettings>;
    OpenStorageFolder(): Promise<void>;
    QuitApp(): Promise<void>;
    RecordSample(
        tagId: string,
        timestamp: number,
        value: number,
    ): Promise<void>;
    SaveCurrentSettings(): Promise<void>;
    SaveSettings(settings: AppSettings): Promise<void>;
    SaveSettingsFile(settings: AppSettings, title: string): Promise<void>;
    SetTreePanelWidth?(width: number): Promise<void>;
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

type BindingAppSettings = Parameters<typeof AppBinding.SaveSettings>[0];

function toAppSettings(raw: BindingAppSettings): AppSettings {
    return {
        ...raw,
        plcLinks: raw.plcLinks ?? [],
        tags: raw.tags ?? [],
        yAxes: raw.yAxes ?? [],
    } as unknown as AppSettings;
}

const wailsBackend: BackendApi = {
    CheckStatus: (plcLinkName) => AppBinding.CheckStatus(plcLinkName),
    ClearHistory: () => AppBinding.ClearHistory(),
    Connect: (plcLinkName, ipAddress, rack, slot) =>
        AppBinding.Connect(plcLinkName, ipAddress, rack, slot),
    Disconnect: (plcLinkName) => AppBinding.Disconnect(plcLinkName),
    DisconnectAll: () => AppBinding.DisconnectAll(),
    ExportCSV: (title) => AppBinding.ExportCSV(title),
    async GetHistoryRange(tagIds, startMs, endMs, maxPoints) {
        const result = await AppBinding.GetHistoryRange(tagIds, startMs, endMs, maxPoints);
        const normalized: Record<string, Array<{ t: number; v: number }>> = {};
        if (result) {
            for (const [key, points] of Object.entries(result)) {
                normalized[key] = points ?? [];
            }
        }
        return normalized;
    },
    GetLastSettingsPath: () => AppBinding.GetLastSettingsPath(),
    async GetSettings() {
        return toAppSettings(await AppBinding.GetSettings());
    },
    GetTreePanelWidth: () => AppBinding.GetTreePanelWidth(),
    GetSystemLanguage: () => AppBinding.GetSystemLanguage(),
    GetVersion: () => AppBinding.GetVersion(),
    HasSettingsChanged: () => AppBinding.HasSettingsChanged(),
    async LoadSettingsFile(title) {
        const loaded = await AppBinding.LoadSettingsFile(title);
        if (!loaded) {
            return loaded as unknown as AppSettings;
        }
        return toAppSettings(loaded);
    },
    OpenStorageFolder: () => AppBinding.OpenStorageFolder(),
    QuitApp: () => AppBinding.QuitApp(),
    RecordSample: (tagId, timestamp, value) =>
        AppBinding.RecordSample(tagId, timestamp, value),
    SaveCurrentSettings: () => AppBinding.SaveCurrentSettings(),
    SaveSettings: (settings) =>
        AppBinding.SaveSettings(settings as unknown as BindingAppSettings),
    SaveSettingsFile: (settings, title) =>
        AppBinding.SaveSettingsFile(settings as unknown as BindingAppSettings, title),
    SetTreePanelWidth: (width) => AppBinding.SetTreePanelWidth(width),
    StartPolling: (settings) =>
        AppBinding.StartPolling(settings as unknown as BindingAppSettings),
    StopPolling: () => AppBinding.StopPolling(),
    TestConnection: (link) => AppBinding.TestConnection(link),
};

const wailsV3Runtime: WailsRuntimeApi = {
    EventsOn: (eventName, callback) =>
        Events.On(eventName, (ev: any) => callback(ev?.data)),
    EventsOff: (eventName, ...additionalEventNames) =>
        Events.Off(eventName, ...additionalEventNames),
    EventsOnce: (eventName, callback) => {
        void Events.Once(eventName, (ev: any) => callback(ev?.data));
    },
    EventsEmit: (eventName, ...optionalData) => {
        if (optionalData.length === 0) {
            void Events.Emit(eventName);
        } else if (optionalData.length === 1) {
            void Events.Emit(eventName, optionalData[0]);
        } else {
            void Events.Emit(eventName, optionalData);
        }
    },
    WindowSetTitle: (title) => {
        void Window.SetTitle(title);
    },
    Quit: () => {
        void Application.Quit();
    },
};

let backendOverride: BackendApi | null = null;
let runtimeOverride: WailsRuntimeApi | null = null;

/** Test hook for injecting a mock BackendApi */
export function setBackendOverride(api: BackendApi | null): void {
    backendOverride = api;
}

/** Test hook for injecting a mock WailsRuntimeApi */
export function setRuntimeOverride(rt: WailsRuntimeApi | null): void {
    runtimeOverride = rt;
}

/** Detects whether the frontend is hosted inside a Wails v3 native webview */
export function isWailsRuntime(): boolean {
    if (typeof window === 'undefined') return false;
    const win = window as any;
    return Boolean(
        win.chrome?.webview?.postMessage ||
            win.webkit?.messageHandlers?.external?.postMessage ||
            win.wails?.invoke ||
            win._wails?.flags,
    );
}

/** Backend API accessor – returns typed Wails v3 Go bindings or undefined in browser preview */
export function backend(): BackendApi | undefined {
    if (backendOverride) return backendOverride;
    if (!isWailsRuntime()) return undefined;
    return wailsBackend;
}

/** Wails runtime accessor */
export function wailsRuntime(): WailsRuntimeApi | undefined {
    if (runtimeOverride) return runtimeOverride;
    if (!isWailsRuntime()) return undefined;
    return wailsV3Runtime;
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
