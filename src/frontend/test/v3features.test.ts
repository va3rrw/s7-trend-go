import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import App from '../src/App.vue';
import { i18n } from '../src/i18n';
import {
    state,
    resetStats,
    backend,
    wailsRuntime,
    isWebServerClient,
    setRuntimeOverride,
    setBackendOverride,
    type TelemetryStream,
} from '../src/store';
import {
    applyLoadedSettings,
    menuLoadSettings,
    menuSaveSettings,
    menuExport,
    menuToggleWebServer,
    menuWebServerPort,
} from '../src/utils/menuActions';
import { defaultSettings, type AppSettings } from '../src/types';
import { createMockBackend, createMockRuntime } from './setup';

describe('Wails v3 features (Streams, Tray sync, File drop settings)', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
        state.isSampling = false;
        state.isPaused = false;
        state.statusMessage = 'Ready';
        resetStats();
    });

    it('applyLoadedSettings updates state.settings and resets sampledRange', () => {
        state.sampledRange['tag-old'] = { min: 1, max: 2 };

        const loaded: AppSettings = {
            pollIntervalMs: 250,
            timeWindowSeconds: 180,
            interpolation: 'Step',
            plcLinks: [
                {
                    name: 'PLC_A',
                    ipAddress: '10.0.0.5',
                    rack: 0,
                    slot: 2,
                    isConnected: false,
                },
            ],
            tags: [
                {
                    id: 'tag-1',
                    name: 'Pressure',
                    plcLink: 'PLC_A',
                    address: 'DB1.DBD0',
                    dataType: 'Real',
                    yAxis: 'Y-Axis 1',
                    color: '#ff0000',
                    enabled: true,
                    samplingIntervalMs: 500,
                },
            ],
            yAxes: [
                {
                    name: 'Y-Axis 1',
                    minimum: 0,
                    maximum: 200,
                    autoScale: false,
                },
            ],
        };

        applyLoadedSettings(loaded);

        expect(state.sampledRange).toEqual({});
        expect(state.settings.pollIntervalMs).toBe(250);
        expect(state.settings.timeWindowSeconds).toBe(180);
        expect(state.settings.interpolation).toBe('Step');
        expect(state.settings.plcLinks[0].name).toBe('PLC_A');
        expect(state.settings.tags).toHaveLength(1);
        expect(state.settings.tags[0].id).toBe('tag-1');
    });

    it('App.vue consumes batched JSONStream telemetry, tray sampling state, and dropped settings events', async () => {
        const listeners = new Map<string, (...data: any[]) => void>();
        const closeMock = vi.fn();
        const streamMock: TelemetryStream = {
            onmessage: null,
            close: closeMock,
        };

        const mockRuntime = createMockRuntime();
        mockRuntime.EventsOn = vi.fn((eventName, cb) => {
            listeners.set(eventName, cb);
            return () => listeners.delete(eventName);
        });
        mockRuntime.OpenJSONStream = vi.fn(() => streamMock);
        setRuntimeOverride(mockRuntime);

        const initial = defaultSettings();
        initial.tags = [
            {
                id: 'tag-good',
                name: 'Temp',
                plcLink: 'PLC1',
                address: 'DB1.DBD0',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                color: '#00ff00',
                enabled: true,
            },
            {
                id: 'tag-bad',
                name: 'Faulted',
                plcLink: 'PLC1',
                address: 'DB1.DBD4',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                color: '#ff0000',
                enabled: true,
            },
        ];

        const mockBackend = createMockBackend();
        mockBackend.GetSettings = vi.fn().mockResolvedValue(initial);
        setBackendOverride(mockBackend);

        const wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        expect(wrapper.find('#app').attributes('data-file-drop-target')).toBeDefined();
        expect(mockRuntime.OpenJSONStream).toHaveBeenCalledWith('polling');

        // 1. Sync sampling state from tray
        listeners.get('app:sampling-state')?.({ isSampling: true });
        await flushPromises();
        expect(state.isSampling).toBe(true);
        expect(state.isPaused).toBe(false);

        // 2. Receive a batched TelemetryFrame with Good + Bad updates and PollTiming
        streamMock.onmessage?.({
            data: {
                updates: [
                    {
                        tagId: 'tag-good',
                        timestamp: '2026-01-01T00:00:00Z',
                        value: '42.5',
                        numericValue: 42.5,
                        quality: 'Good',
                        error: '',
                    },
                    {
                        tagId: 'tag-bad',
                        timestamp: '2026-01-01T00:00:00Z',
                        value: '',
                        numericValue: null,
                        quality: 'Bad',
                        error: 'timeout',
                    },
                ],
                timings: [{ plcLink: 'PLC1', actualIntervalMs: 37 }],
            },
        });
        await flushPromises();

        expect(state.sampledRange['tag-good']).toEqual({ min: 42.5, max: 42.5 });
        // Bad update with null numericValue must NOT record 0 into sampledRange
        expect(state.sampledRange['tag-bad']).toBeUndefined();
        expect(wrapper.text()).toContain('PLC1: 37ms');

        // 3. Receive dropped settings loaded & error events
        const dropped: AppSettings = {
            ...initial,
            pollIntervalMs: 500,
            interpolation: 'Step',
        };
        listeners.get('settings:loaded')?.(dropped);
        await flushPromises();
        expect(state.settings.pollIntervalMs).toBe(500);
        expect(state.settings.interpolation).toBe('Step');
        expect(state.sampledRange).toEqual({});

        listeners.get('settings:load-error')?.('bad json');
        await flushPromises();
        expect(state.statusMessage).toContain('bad json');

        // 4. Tray stop resets sampling & clears actual timing readout
        listeners.get('app:sampling-state')?.({ isSampling: false });
        await flushPromises();
        expect(state.isSampling).toBe(false);
        expect(wrapper.text()).not.toContain('PLC1: 37ms');

        wrapper.unmount();
        expect(closeMock).toHaveBeenCalledTimes(1);
    });

    it('detects web server client, supports cancellable HTTP GetHistoryRange, and streams WebSocket polling/events', async () => {
        setBackendOverride(null);
        setRuntimeOverride(null);

        expect(isWebServerClient()).toBe(false);
        expect(backend()).toBeUndefined();
        expect(wailsRuntime()).toBeUndefined();

        (window as any).__S7_WEB_SERVER__ = true;
        expect(isWebServerClient()).toBe(true);

        const api = backend();
        const rt = wailsRuntime();
        expect(api).toBeDefined();
        expect(rt).toBeDefined();

        // 1. Cancellable GetHistoryRange aborts underlying fetch AbortSignal
        const originalFetch = globalThis.fetch;
        let capturedSignal: AbortSignal | undefined;
        globalThis.fetch = vi.fn((_input: any, init?: RequestInit) => {
            capturedSignal = init?.signal as AbortSignal | undefined;
            return new Promise<Response>((_resolve, reject) => {
                capturedSignal?.addEventListener('abort', () => {
                    reject(Object.assign(new Error('Aborted'), { name: 'AbortError' }));
                });
            });
        }) as any;

        try {
            const pending = api!.GetHistoryRange(['tag-1'], 0, 1000, 100);
            expect(capturedSignal?.aborted).toBe(false);
            pending.cancel?.();
            expect(capturedSignal?.aborted).toBe(true);
            await expect(pending).rejects.toThrow('Aborted');
        } finally {
            globalThis.fetch = originalFetch;
        }

        // 2. WebSocket telemetry stream and event hub
        class FakeWebSocket {
            static instances: FakeWebSocket[] = [];
            static OPEN = 1;
            static CONNECTING = 0;
            url: string;
            readyState = FakeWebSocket.OPEN;
            onmessage: ((ev: { data: any }) => void) | null = null;
            onerror: ((ev: any) => void) | null = null;
            onclose: ((ev: any) => void) | null = null;
            closed = false;

            constructor(url: string) {
                this.url = url;
                FakeWebSocket.instances.push(this);
            }
            close() {
                this.closed = true;
                this.readyState = 3;
            }
        }

        const originalWS = globalThis.WebSocket;
        globalThis.WebSocket = FakeWebSocket as any;

        try {
            const receivedFrames: any[] = [];
            const stream = rt!.OpenJSONStream!('polling');
            stream.onmessage = (ev) => receivedFrames.push(ev.data);

            const pollWs = FakeWebSocket.instances[0];
            expect(pollWs.url).toContain('/api/ws/polling');
            pollWs.onmessage?.({ data: JSON.stringify({ updates: [{ tagId: 't1', value: '12.3' }] }) });
            expect(receivedFrames).toEqual([{ updates: [{ tagId: 't1', value: '12.3' }] }]);
            stream.close();
            expect(pollWs.closed).toBe(true);

            const receivedEvents: any[] = [];
            const off = rt!.EventsOn('app:web-server-state', (data) => receivedEvents.push(data));
            const eventWs = FakeWebSocket.instances[1];
            expect(eventWs.url).toContain('/api/ws/events');
            eventWs.onmessage?.({
                data: JSON.stringify({ name: 'app:web-server-state', data: { running: true, port: 9090 } }),
            });
            expect(receivedEvents).toEqual([{ running: true, port: 9090 }]);
            off();
            expect(eventWs.closed).toBe(true);
        } finally {
            globalThis.WebSocket = originalWS;
        }
    });

    it('handles desktop Web Server menu actions and browser import/save/export paths', async () => {
        const t = (key: string, args?: any[]) => (args ? `${key}:${args.join(',')}` : key);
        const mockBackend = createMockBackend();
        mockBackend.ToggleWebServer = vi.fn().mockResolvedValue({
            enabled: true,
            running: true,
            host: '0.0.0.0',
            port: 8080,
            url: 'http://192.168.1.5:8080',
            connectedClients: 0,
        });
        mockBackend.SetWebServerPort = vi.fn().mockResolvedValue({
            enabled: true,
            running: true,
            host: '0.0.0.0',
            port: 9090,
            url: 'http://192.168.1.5:9090',
            connectedClients: 0,
        });
        setBackendOverride(mockBackend);

        await menuToggleWebServer(t);
        expect(mockBackend.ToggleWebServer).toHaveBeenCalledTimes(1);
        expect(state.webServer.running).toBe(true);
        expect(state.statusMessage).toContain('status.web_server_started');

        const showMsg = vi.fn().mockResolvedValue(true);
        await menuWebServerPort(t, vi.fn().mockResolvedValue('80'), showMsg);
        expect(showMsg).toHaveBeenCalledWith('menu.web_server_port', 'prompt.invalid_port');
        expect(mockBackend.SetWebServerPort).not.toHaveBeenCalled();

        await menuWebServerPort(t, vi.fn().mockResolvedValue('9090'), showMsg);
        expect(mockBackend.SetWebServerPort).toHaveBeenCalledWith(9090);
        expect(state.webServer.port).toBe(9090);

        // Browser file download & JSON import paths
        (window as any).__S7_WEB_SERVER__ = true;
        const imported = defaultSettings();
        imported.pollIntervalMs = 750;
        mockBackend.ImportSettingsJSON = vi.fn().mockResolvedValue(imported);

        const clickedLinks: Array<{ href: string; download: string }> = [];
        const origClick = HTMLAnchorElement.prototype.click;
        HTMLAnchorElement.prototype.click = function (this: HTMLAnchorElement) {
            clickedLinks.push({ href: this.getAttribute('href') || '', download: this.download });
        };

        const origCreateObjURL = URL.createObjectURL;
        const origRevokeObjURL = URL.revokeObjectURL;
        URL.createObjectURL = vi.fn(() => 'blob:mock-settings');
        URL.revokeObjectURL = vi.fn();

        const origInputClick = HTMLInputElement.prototype.click;
        HTMLInputElement.prototype.click = function (this: HTMLInputElement) {
            const file = new File([JSON.stringify(imported)], 'cfg.json', { type: 'application/json' });
            Object.defineProperty(this, 'files', { value: [file], configurable: true });
            this.dispatchEvent(new Event('change'));
        };

        try {
            await menuSaveSettings(t);
            await menuExport(t);
            expect(clickedLinks).toEqual([
                { href: 'blob:mock-settings', download: 's7-trend-settings.json' },
                { href: '/api/export/csv', download: 'trend.csv' },
            ]);
            expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock-settings');

            await menuLoadSettings(t);
            expect(mockBackend.ImportSettingsJSON).toHaveBeenCalledTimes(1);
            expect(state.settings.pollIntervalMs).toBe(750);
            expect(state.statusMessage).toBe('status.loaded_settings');
        } finally {
            HTMLAnchorElement.prototype.click = origClick;
            HTMLInputElement.prototype.click = origInputClick;
            URL.createObjectURL = origCreateObjURL;
            URL.revokeObjectURL = origRevokeObjURL;
        }
    });

    it('App.vue syncs IsPolling + GetWebServerStatus, sets (Web) title, and deduplicates app:sampling-state', async () => {
        (window as any).__S7_WEB_SERVER__ = true;
        const listeners = new Map<string, (...data: any[]) => void>();
        const mockRuntime = createMockRuntime();
        mockRuntime.EventsOn = vi.fn((eventName, cb) => {
            listeners.set(eventName, cb);
            return () => listeners.delete(eventName);
        });
        setRuntimeOverride(mockRuntime);

        const mockBackend = createMockBackend();
        mockBackend.IsPolling = vi.fn().mockResolvedValue(true);
        mockBackend.GetWebServerStatus = vi.fn().mockResolvedValue({
            enabled: true,
            running: true,
            host: '0.0.0.0',
            port: 8080,
            url: 'http://127.0.0.1:8080',
            connectedClients: 1,
        });
        setBackendOverride(mockBackend);

        const wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        expect(state.isSampling).toBe(true);
        expect(state.webServer.running).toBe(true);
        expect(mockRuntime.WindowSetTitle).toHaveBeenCalledWith('S7 Trend Go (Web)');

        // Deduplicate identical app:sampling-state without clearing pause state
        state.isPaused = true;
        listeners.get('app:sampling-state')?.({ isSampling: true });
        await flushPromises();
        expect(state.isPaused).toBe(true);

        // Live app:web-server-state update
        listeners.get('app:web-server-state')?.({
            enabled: true,
            running: true,
            host: '0.0.0.0',
            port: 9091,
            url: 'http://127.0.0.1:9091',
            connectedClients: 2,
        });
        await flushPromises();
        expect(state.webServer.port).toBe(9091);
        expect(state.webServer.connectedClients).toBe(2);

        wrapper.unmount();
    });

    it('httpRuntime.OpenJSONStream automatically reconnects after unexpected close and stops cleanly on close()', () => {
        (window as any).__S7_WEB_SERVER__ = true;
        setRuntimeOverride(null);

        class ReconnectingFakeWebSocket {
            static instances: ReconnectingFakeWebSocket[] = [];
            static OPEN = 1;
            static CONNECTING = 0;
            url: string;
            readyState = ReconnectingFakeWebSocket.OPEN;
            onmessage: ((ev: { data: any }) => void) | null = null;
            onerror: ((ev: any) => void) | null = null;
            onclose: ((ev: any) => void) | null = null;
            closed = false;

            constructor(url: string) {
                this.url = url;
                ReconnectingFakeWebSocket.instances.push(this);
            }
            close() {
                this.closed = true;
                this.readyState = 3;
            }
        }

        const originalWS = globalThis.WebSocket;
        globalThis.WebSocket = ReconnectingFakeWebSocket as any;
        vi.useFakeTimers();

        try {
            const received: any[] = [];
            const stream = wailsRuntime()!.OpenJSONStream!('polling');
            stream.onmessage = (ev) => received.push(ev.data);

            expect(ReconnectingFakeWebSocket.instances).toHaveLength(1);
            const firstWs = ReconnectingFakeWebSocket.instances[0];

            // Simulate unexpected disconnect (e.g. network blip or backend write timeout)
            firstWs.onclose?.({} as any);
            vi.advanceTimersByTime(1500);

            expect(ReconnectingFakeWebSocket.instances).toHaveLength(2);
            const secondWs = ReconnectingFakeWebSocket.instances[1];
            expect(secondWs.url).toContain('/api/ws/polling');

            secondWs.onmessage?.({
                data: JSON.stringify({ updates: [{ tagId: 't-fast', value: '99.5' }] }),
            });
            expect(received).toEqual([{ updates: [{ tagId: 't-fast', value: '99.5' }] }]);

            // Explicit close must not schedule any further reconnect
            stream.close();
            expect(secondWs.closed).toBe(true);
            secondWs.onclose?.({} as any);
            vi.advanceTimersByTime(3000);
            expect(ReconnectingFakeWebSocket.instances).toHaveLength(2);
        } finally {
            vi.useRealTimers();
            globalThis.WebSocket = originalWS;
        }
    });
});
