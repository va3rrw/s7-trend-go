import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import App from '../src/App.vue';
import { i18n } from '../src/i18n';
import {
    state,
    resetStats,
    setRuntimeOverride,
    setBackendOverride,
    type TelemetryStream,
} from '../src/store';
import { applyLoadedSettings } from '../src/utils/menuActions';
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
});
