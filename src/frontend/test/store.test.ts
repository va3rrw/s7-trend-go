import { describe, it, expect, beforeEach } from 'vitest';
import {
    state,
    uiState,
    showMessage,
    backend,
    wailsRuntime,
    formatNumber,
    updateTagValue,
    resetStats,
    getTag,
    markPlcForTag,
} from '../src/store';
import { defaultSettings } from '../src/types';

describe('store.ts', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
        state.isSampling = false;
        state.isPaused = false;
        state.statusMessage = 'Ready';
        state.recordsHistory = [];
        resetStats();

        uiState.msgOpen = false;
        uiState.msgTitle = '';
        uiState.msgMessage = '';
        uiState.msgShowCancel = false;
        uiState.msgResolve = null;
    });

    it('formats numbers properly without extra trailing zeroes', () => {
        expect(formatNumber(10)).toBe('10');
        expect(formatNumber(0)).toBe('0');
        expect(formatNumber(12.3456)).toBe('12.346');
        expect(formatNumber(12.3)).toBe('12.3');
        expect(formatNumber(12.0)).toBe('12');
    });

    it('tracks min and max values with updateTagValue and resetStats', () => {
        updateTagValue('tag-1', '10.5', 10.5);
        updateTagValue('tag-1', '25.0', 25.0);
        updateTagValue('tag-1', '5.2', 5.2);

        const range = state.sampledRange.get('tag-1');
        expect(range).toBeDefined();
        expect(range?.min).toBe(5.2);
        expect(range?.max).toBe(25.0);

        // Undefined or NaN should not break the range
        updateTagValue('tag-1', 'NaN', NaN);
        expect(state.sampledRange.get('tag-1')?.min).toBe(5.2);

        // Reset clears all stats
        resetStats();
        expect(state.sampledRange.size).toBe(0);
    });

    it('finds tag by ID with getTag()', () => {
        const tag = {
            id: 'test-uuid-123',
            name: 'Speed',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            stringLength: 0,
            yAxis: 'Y-Axis 1',
            lowLimit: 0,
            highLimit: 100,
            color: '#FF0000',
            enabled: true,
            notification: 'None',
        };
        state.settings.tags.push(tag);

        expect(getTag('test-uuid-123')).toEqual(tag);
        expect(getTag('non-existent')).toBeUndefined();
    });

    it('updates PLC connection status with markPlcForTag()', () => {
        const tag = {
            id: 'tag-1',
            name: 'Speed',
            plcLink: 'PLC2',
            address: 'DB1.DBD0',
            dataType: 'Real',
            stringLength: 0,
            yAxis: 'Y-Axis 1',
            lowLimit: 0,
            highLimit: 100,
            color: '#FF0000',
            enabled: true,
            notification: 'None',
        };
        state.settings.tags.push(tag);

        markPlcForTag('tag-1', 'Good');
        const link = state.settings.plcLinks.find((p) => p.name === 'PLC2');
        expect(link?.isConnected).toBe(true);

        markPlcForTag('tag-1', 'Bad');
        expect(link?.isConnected).toBe(false);

        // Unknown tag should be ignored safely
        markPlcForTag('unknown-tag', 'Good');
    });

    it('controls UI dialog state with showMessage() and promise resolve', async () => {
        const promise = showMessage('Warning', 'Something happened', true);

        expect(uiState.msgOpen).toBe(true);
        expect(uiState.msgTitle).toBe('Warning');
        expect(uiState.msgMessage).toBe('Something happened');
        expect(uiState.msgShowCancel).toBe(true);
        expect(uiState.msgResolve).toBeInstanceOf(Function);

        // Simulate user clicking OK
        uiState.msgResolve?.(true);
        const result = await promise;
        expect(result).toBe(true);
    });

    it('provides backend and wailsRuntime accessors', () => {
        expect(backend()).toBeDefined();
        expect(wailsRuntime()).toBeDefined();
    });
});
