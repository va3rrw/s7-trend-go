import { reactive } from 'vue';
import { Application, Events, JSONStream, Window } from '@wailsio/runtime';
import * as AppBinding from '../bindings/s7-trend-go/backend/app';
import type { AppSettings, TagSettings } from './types';
import { defaultSettings } from './types';

export type MaybeCancellablePromise<T> = Promise<T> & {
    cancel?: (cause?: any) => void | PromiseLike<void>;
};

export interface TelemetryStream {
    onmessage: ((ev: { data: any }) => void) | null;
    onerror?: ((ev: any) => void) | null;
    onclose?: ((ev: any) => void) | null;
    close(): void;
}

export interface WebServerStatus {
    enabled: boolean;
    running: boolean;
    host: string;
    port: number;
    url: string;
    error?: string;
    connectedClients: number;
}

/** Reactive application state shared across all components */
export const state = reactive({
    settings: defaultSettings() as AppSettings,
    isSampling: false,
    isPaused: false,
    statusMessage: 'Ready',
    sampledRange: {} as Record<string, { min: number; max: number }>,
    webServer: {
        enabled: false,
        running: false,
        host: '0.0.0.0',
        port: 8080,
        url: '',
        connectedClients: 0,
    } as WebServerStatus,
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
    ): MaybeCancellablePromise<Record<string, Array<{ t: number; v: number }>>>;
    GetLastSettingsPath(): Promise<string>;
    GetSettings(): Promise<AppSettings>;
    GetTreePanelWidth?(): Promise<number>;
    GetSystemLanguage(): Promise<string>;
    GetVersion(): Promise<string>;
    GetWebServerStatus?(): Promise<WebServerStatus>;
    HasSettingsChanged(): Promise<boolean>;
    ImportSettingsJSON?(rawJSON: string): Promise<AppSettings>;
    IsPolling?(): Promise<boolean>;
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
    SetWebServerPort?(port: number): Promise<WebServerStatus>;
    StartPolling(settings: AppSettings): Promise<void>;
    StartWebServer?(host: string, port: number): Promise<WebServerStatus>;
    StopPolling(): Promise<void>;
    StopWebServer?(): Promise<WebServerStatus>;
    TestConnection(link: {
        name: string;
        ipAddress: string;
        rack: number;
        slot: number;
        isConnected: boolean;
    }): Promise<void>;
    ToggleWebServer?(): Promise<WebServerStatus>;
}

export interface WailsRuntimeApi {
    EventsOn(eventName: string, callback: (...data: any) => void): () => void;
    EventsOff(eventName: string, ...additionalEventNames: string[]): void;
    EventsOnce(eventName: string, callback: (...data: any) => void): void;
    EventsEmit(eventName: string, ...optionalData: any): void;
    OpenJSONStream?(name: string): TelemetryStream;
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
    GetHistoryRange(tagIds, startMs, endMs, maxPoints) {
        return AppBinding.GetHistoryRange(tagIds, startMs, endMs, maxPoints).then(
            (result) => {
                const normalized: Record<string, Array<{ t: number; v: number }>> = {};
                if (result) {
                    for (const [key, points] of Object.entries(result)) {
                        normalized[key] = points ?? [];
                    }
                }
                return normalized;
            },
        );
    },
    GetLastSettingsPath: () => AppBinding.GetLastSettingsPath(),
    async GetSettings() {
        return toAppSettings(await AppBinding.GetSettings());
    },
    GetTreePanelWidth: () => AppBinding.GetTreePanelWidth(),
    GetSystemLanguage: () => AppBinding.GetSystemLanguage(),
    GetVersion: () => AppBinding.GetVersion(),
    GetWebServerStatus: () => AppBinding.GetWebServerStatus(),
    HasSettingsChanged: () => AppBinding.HasSettingsChanged(),
    async ImportSettingsJSON(rawJSON) {
        const loaded = await AppBinding.ImportSettingsJSON(rawJSON);
        if (!loaded) {
            return loaded as unknown as AppSettings;
        }
        return toAppSettings(loaded);
    },
    IsPolling: () => AppBinding.IsPolling(),
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
    SetWebServerPort: (port) => AppBinding.SetWebServerPort(port),
    StartPolling: (settings) =>
        AppBinding.StartPolling(settings as unknown as BindingAppSettings),
    StartWebServer: (host, port) => AppBinding.StartWebServer(host, port),
    StopPolling: () => AppBinding.StopPolling(),
    StopWebServer: () => AppBinding.StopWebServer(),
    TestConnection: (link) => AppBinding.TestConnection(link),
    ToggleWebServer: () => AppBinding.ToggleWebServer(),
};

function buildWsUrl(path: string): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${proto}//${window.location.host}${path}`;
}

async function postRPC<T>(
    method: string,
    body?: unknown,
    signal?: AbortSignal,
): Promise<T> {
    const res = await fetch(`/api/rpc/${encodeURIComponent(method)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body !== undefined ? JSON.stringify(body) : '{}',
        signal,
    });
    if (!res.ok) {
        let message = `${res.status} ${res.statusText}`.trim();
        try {
            const payload = await res.json();
            if (payload && typeof payload.error === 'string' && payload.error) {
                message = payload.error;
            }
        } catch {
            // keep HTTP status text
        }
        throw new Error(message);
    }
    const text = await res.text();
    if (!text) return undefined as T;
    return JSON.parse(text) as T;
}

const httpBackend: BackendApi = {
    CheckStatus: (plcLinkName) =>
        postRPC<boolean>('CheckStatus', { linkName: plcLinkName }),
    ClearHistory: () => postRPC<void>('ClearHistory'),
    Connect: (plcLinkName, ipAddress, rack, slot) =>
        postRPC<void>('Connect', {
            linkName: plcLinkName,
            ip: ipAddress,
            rack,
            slot,
        }),
    Disconnect: (plcLinkName) =>
        postRPC<void>('Disconnect', { linkName: plcLinkName }),
    DisconnectAll: () => postRPC<void>('DisconnectAll'),
    ExportCSV: async () => {
        if (typeof window !== 'undefined') {
            window.location.assign('/api/export/csv');
        }
    },
    GetHistoryRange(tagIds, startMs, endMs, maxPoints) {
        const controller = new AbortController();
        const promise = postRPC<Record<string, Array<{ t: number; v: number }> | null>>(
            'GetHistoryRange',
            { tagIds, startMs, endMs, maxPoints },
            controller.signal,
        ).then((result) => {
            const normalized: Record<string, Array<{ t: number; v: number }>> = {};
            if (result) {
                for (const [key, points] of Object.entries(result)) {
                    normalized[key] = points ?? [];
                }
            }
            return normalized;
        }) as MaybeCancellablePromise<
            Record<string, Array<{ t: number; v: number }>>
        >;
        promise.cancel = () => controller.abort();
        return promise;
    },
    GetLastSettingsPath: () => postRPC<string>('GetLastSettingsPath'),
    async GetSettings() {
        return toAppSettings(await postRPC<BindingAppSettings>('GetSettings'));
    },
    GetTreePanelWidth: () => postRPC<number>('GetTreePanelWidth'),
    GetSystemLanguage: () => postRPC<string>('GetSystemLanguage'),
    GetVersion: () => postRPC<string>('GetVersion'),
    GetWebServerStatus: () => postRPC<WebServerStatus>('GetWebServerStatus'),
    HasSettingsChanged: () => postRPC<boolean>('HasSettingsChanged'),
    async ImportSettingsJSON(rawJSON) {
        const loaded = await postRPC<BindingAppSettings>('ImportSettingsJSON', {
            rawJSON,
        });
        if (!loaded) {
            return loaded as unknown as AppSettings;
        }
        return toAppSettings(loaded);
    },
    IsPolling: () => postRPC<boolean>('IsPolling'),
    LoadSettingsFile: () =>
        Promise.reject(new Error('Native file dialog is unavailable in web client')),
    OpenStorageFolder: () =>
        Promise.reject(new Error('OpenStorageFolder is unavailable in web client')),
    QuitApp: () => Promise.resolve(),
    RecordSample: (tagId, timestamp, value) =>
        postRPC<void>('RecordSample', {
            tagId,
            timestampMs: timestamp,
            value,
        }),
    SaveCurrentSettings: () =>
        Promise.reject(new Error('Native file dialog is unavailable in web client')),
    SaveSettings: (settings) => postRPC<void>('SaveSettings', settings),
    SaveSettingsFile: () =>
        Promise.reject(new Error('Native file dialog is unavailable in web client')),
    SetTreePanelWidth: (width) =>
        postRPC<void>('SetTreePanelWidth', { width }),
    StartPolling: (settings) => postRPC<void>('StartPolling', settings),
    StopPolling: () => postRPC<void>('StopPolling'),
    TestConnection: (link) => postRPC<void>('TestConnection', link),
};

const httpEventListeners = new Map<string, Set<(...data: any) => void>>();
let httpEventSocket: WebSocket | null = null;
let httpEventReconnectTimer: number | null = null;

function countHttpListeners(): number {
    let total = 0;
    for (const set of httpEventListeners.values()) {
        total += set.size;
    }
    return total;
}

function closeIdleHttpEventSocket(): void {
    if (countHttpListeners() > 0) return;
    if (httpEventReconnectTimer !== null && typeof window !== 'undefined') {
        window.clearTimeout(httpEventReconnectTimer);
        httpEventReconnectTimer = null;
    }
    if (httpEventSocket) {
        const sock = httpEventSocket;
        httpEventSocket = null;
        try {
            sock.close();
        } catch {
            // ignore close error
        }
    }
}

function ensureHttpEventSocket(): void {
    if (typeof window === 'undefined' || typeof WebSocket === 'undefined') {
        return;
    }
    if (
        httpEventSocket &&
        (httpEventSocket.readyState === WebSocket.OPEN ||
            httpEventSocket.readyState === WebSocket.CONNECTING)
    ) {
        return;
    }

    const ws = new WebSocket(buildWsUrl('/api/ws/events'));
    httpEventSocket = ws;

    ws.onmessage = (ev) => {
        if (typeof ev.data !== 'string') return;
        let parsed: { name?: string; data?: any } | null = null;
        try {
            parsed = JSON.parse(ev.data);
        } catch {
            return;
        }
        if (!parsed?.name) return;
        const listeners = httpEventListeners.get(parsed.name);
        if (!listeners || listeners.size === 0) return;
        for (const cb of Array.from(listeners)) {
            cb(parsed.data);
        }
    };

    ws.onclose = () => {
        if (httpEventSocket === ws) {
            httpEventSocket = null;
        }
        if (countHttpListeners() > 0 && httpEventReconnectTimer === null) {
            httpEventReconnectTimer = window.setTimeout(() => {
                httpEventReconnectTimer = null;
                if (countHttpListeners() > 0) {
                    ensureHttpEventSocket();
                }
            }, 1500);
        }
    };
}

const httpRuntime: WailsRuntimeApi = {
    EventsOn: (eventName, callback) => {
        let listeners = httpEventListeners.get(eventName);
        if (!listeners) {
            listeners = new Set();
            httpEventListeners.set(eventName, listeners);
        }
        listeners.add(callback);
        ensureHttpEventSocket();
        return () => {
            listeners?.delete(callback);
            if (listeners && listeners.size === 0) {
                httpEventListeners.delete(eventName);
            }
            closeIdleHttpEventSocket();
        };
    },
    EventsOff: (eventName, ...additionalEventNames) => {
        for (const name of [eventName, ...additionalEventNames]) {
            httpEventListeners.delete(name);
        }
        closeIdleHttpEventSocket();
    },
    EventsOnce: (eventName, callback) => {
        const off = httpRuntime.EventsOn(eventName, (...args: any) => {
            off();
            callback(...args);
        });
    },
    EventsEmit: () => {
        /* local-only in browser client */
    },
    OpenJSONStream: (name) => {
        const wsPath = `/api/ws/${encodeURIComponent(name)}`;
        let ws: WebSocket | null = null;
        let reconnectTimer: number | null = null;
        let closed = false;

        const stream: TelemetryStream = {
            onmessage: null,
            onerror: null,
            onclose: null,
            close() {
                if (closed) return;
                closed = true;
                if (reconnectTimer !== null && typeof window !== 'undefined') {
                    window.clearTimeout(reconnectTimer);
                    reconnectTimer = null;
                }
                const sock = ws;
                ws = null;
                if (sock) {
                    try {
                        sock.close();
                    } catch {
                        // ignore close errors
                    }
                }
            },
        };

        const connect = () => {
            if (closed || typeof WebSocket === 'undefined' || typeof window === 'undefined') {
                return;
            }
            const sock = new WebSocket(buildWsUrl(wsPath));
            ws = sock;
            sock.onmessage = (ev) => {
                if (closed) return;
                let data: any = ev.data;
                if (typeof data === 'string') {
                    try {
                        data = JSON.parse(data);
                    } catch {
                        // pass through raw string
                    }
                }
                stream.onmessage?.({ data });
            };
            sock.onerror = (ev) => {
                if (!closed) {
                    stream.onerror?.(ev);
                }
            };
            sock.onclose = (ev) => {
                if (ws === sock) {
                    ws = null;
                }
                stream.onclose?.(ev);
                if (!closed && reconnectTimer === null) {
                    reconnectTimer = window.setTimeout(() => {
                        reconnectTimer = null;
                        connect();
                    }, 1500);
                }
            };
        };

        connect();
        return stream;
    },
    WindowSetTitle: (title) => {
        if (typeof document !== 'undefined') {
            document.title = title;
        }
    },
    Quit: () => {
        /* desktop-only */
    },
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
    OpenJSONStream: (name) => JSONStream(name) as unknown as TelemetryStream,
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

/** Detects whether the frontend is running inside the desktop's embedded web server */
export function isWebServerClient(): boolean {
    return (
        typeof window !== 'undefined' &&
        Boolean((window as any).__S7_WEB_SERVER__)
    );
}

/** Backend API accessor – returns Wails v3 bindings, embedded web RPC client, or undefined in Vite preview */
export function backend(): BackendApi | undefined {
    if (backendOverride) return backendOverride;
    if (isWebServerClient()) return httpBackend;
    if (!isWailsRuntime()) return undefined;
    return wailsBackend;
}

/** Wails runtime accessor */
export function wailsRuntime(): WailsRuntimeApi | undefined {
    if (runtimeOverride) return runtimeOverride;
    if (isWebServerClient()) return httpRuntime;
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
