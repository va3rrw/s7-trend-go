import { beforeEach, vi } from 'vitest';
import {
    setBackendOverride,
    setRuntimeOverride,
    type BackendApi,
    type WailsRuntimeApi,
} from '../src/store';
import { defaultSettings } from '../src/types';

// Mock ResizeObserver for Chart.js
if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class ResizeObserver {
        observe = vi.fn();
        unobserve = vi.fn();
        disconnect = vi.fn();
    } as any;
}

// Mock dialog APIs for jsdom
if (typeof HTMLDialogElement !== 'undefined') {
    HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
        this.setAttribute('open', '');
    });
    HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
        this.removeAttribute('open');
    });
}

// Mock canvas getContext for Chart.js
if (typeof HTMLCanvasElement !== 'undefined') {
    HTMLCanvasElement.prototype.getContext = function (this: HTMLCanvasElement, contextId: string) {
        if (contextId === '2d') {
            return {
                canvas: this,
                save: vi.fn(),
                restore: vi.fn(),
                beginPath: vi.fn(),
                moveTo: vi.fn(),
                lineTo: vi.fn(),
                stroke: vi.fn(),
                arc: vi.fn(),
                fill: vi.fn(),
                fillRect: vi.fn(),
                clearRect: vi.fn(),
                getImageData: vi.fn(),
                putImageData: vi.fn(),
                createImageData: vi.fn(),
                setTransform: vi.fn(),
                resetTransform: vi.fn(),
                drawImage: vi.fn(),
                fillText: vi.fn(),
                measureText: vi.fn(() => ({ width: 0 })),
                transform: vi.fn(),
                rect: vi.fn(),
                clip: vi.fn(),
                setLineDash: vi.fn(),
            } as any;
        }
        return null;
    } as any;
}

export function createMockRuntime(): WailsRuntimeApi {
    return {
        EventsOn: vi.fn(() => vi.fn()),
        EventsOff: vi.fn(),
        EventsOnce: vi.fn(),
        EventsEmit: vi.fn(),
        WindowSetTitle: vi.fn(),
        Quit: vi.fn(),
    };
}

export function createMockBackend(): BackendApi {
    return {
        CheckStatus: vi.fn().mockResolvedValue(false),
        ClearHistory: vi.fn().mockResolvedValue(undefined),
        Connect: vi.fn().mockResolvedValue(undefined),
        Disconnect: vi.fn().mockResolvedValue(undefined),
        DisconnectAll: vi.fn().mockResolvedValue(undefined),
        ExportCSV: vi.fn().mockResolvedValue(undefined),
        GetHistoryRange: vi.fn().mockResolvedValue({}),
        GetLastSettingsPath: vi.fn().mockResolvedValue(''),
        GetSettings: vi.fn().mockResolvedValue(defaultSettings()),
        GetTreePanelWidth: vi.fn().mockResolvedValue(250),
        GetSystemLanguage: vi.fn().mockResolvedValue('en'),
        GetVersion: vi.fn().mockResolvedValue('1.0.0'),
        HasSettingsChanged: vi.fn().mockResolvedValue(false),
        LoadSettingsFile: vi.fn().mockResolvedValue(defaultSettings()),
        OpenStorageFolder: vi.fn().mockResolvedValue(undefined),
        QuitApp: vi.fn().mockResolvedValue(undefined),
        RecordSample: vi.fn().mockResolvedValue(undefined),
        SaveCurrentSettings: vi.fn().mockResolvedValue(undefined),
        SaveSettings: vi.fn().mockResolvedValue(undefined),
        SaveSettingsFile: vi.fn().mockResolvedValue(undefined),
        SetTreePanelWidth: vi.fn().mockResolvedValue(undefined),
        StartPolling: vi.fn().mockResolvedValue(undefined),
        StopPolling: vi.fn().mockResolvedValue(undefined),
        TestConnection: vi.fn().mockResolvedValue(undefined),
    };
}

setRuntimeOverride(createMockRuntime());
setBackendOverride(createMockBackend());

beforeEach(() => {
    setRuntimeOverride(createMockRuntime());
    setBackendOverride(createMockBackend());
    vi.clearAllMocks();
});
