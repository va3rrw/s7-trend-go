import { beforeEach, vi } from 'vitest';

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

// Mock Wails runtime & backend on window
(globalThis as any).window = globalThis.window || {};
(window as any).runtime = {
    EventsOn: vi.fn(),
    EventsOff: vi.fn(),
    EventsEmit: vi.fn(),
};

(window as any).go = {
    backend: {
        App: {
            GetSettings: vi.fn().mockResolvedValue({}),
            SaveSettings: vi.fn().mockResolvedValue(undefined),
            StartPolling: vi.fn().mockResolvedValue(undefined),
            StopPolling: vi.fn().mockResolvedValue(undefined),
            TestConnection: vi.fn().mockResolvedValue(undefined),
            SavePlcLinks: vi.fn().mockResolvedValue(undefined),
            LoadPlcLinks: vi.fn().mockResolvedValue([]),
            SaveTags: vi.fn().mockResolvedValue(undefined),
            LoadTags: vi.fn().mockResolvedValue([]),
            ExportCSV: vi.fn().mockResolvedValue(undefined),
            GetHistoryRange: vi.fn().mockResolvedValue({}),
            ClearHistory: vi.fn().mockResolvedValue(undefined),
            GetVersion: vi.fn().mockResolvedValue('1.0.0'),
            GetSystemLanguage: vi.fn().mockResolvedValue('en'),
        },
    },
};

beforeEach(() => {
    vi.clearAllMocks();
});
