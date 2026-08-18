import { describe, it, expect } from 'vitest';
import {
    newId,
    defaultSettings,
    DATA_TYPES,
    NOTIFICATIONS,
    PALETTE,
} from '../src/types';

describe('types.ts', () => {
    it('generates a unique non-empty ID with newId()', () => {
        const id1 = newId();
        const id2 = newId();
        expect(id1).toBeTruthy();
        expect(id2).toBeTruthy();
        expect(id1).not.toBe(id2);
    });

    it('returns valid default application settings', () => {
        const settings = defaultSettings();
        expect(settings.pollIntervalMs).toBe(100);
        expect(settings.timeWindowSeconds).toBe(60);
        expect(settings.interpolation).toBe('Line');
        expect(settings.tags).toEqual([]);
        expect(settings.plcLinks).toHaveLength(4);
        expect(settings.yAxes).toHaveLength(4);

        expect(settings.plcLinks[0]).toEqual({
            name: 'PLC1',
            ipAddress: '192.168.0.1',
            rack: 0,
            slot: 1,
            isConnected: false,
        });

        expect(settings.yAxes[0]).toEqual({
            name: 'Y-Axis 1',
            minimum: 0,
            maximum: 100,
            autoScale: true,
        });
    });

    it('contains expected data types', () => {
        expect(DATA_TYPES).toContain('Bool');
        expect(DATA_TYPES).toContain('Byte');
        expect(DATA_TYPES).toContain('Word');
        expect(DATA_TYPES).toContain('Int');
        expect(DATA_TYPES).toContain('DWord');
        expect(DATA_TYPES).toContain('DInt');
        expect(DATA_TYPES).toContain('Real');
        expect(DATA_TYPES).toContain('LReal');
        expect(DATA_TYPES).toContain('String');
        expect(DATA_TYPES).toHaveLength(9);
    });

    it('contains expected notification options', () => {
        expect(NOTIFICATIONS).toContain('None');
        expect(NOTIFICATIONS).toContain('BinaryPositiveEdge');
        expect(NOTIFICATIONS).toContain('BinaryNegativeEdge');
        expect(NOTIFICATIONS).toContain('BinaryBothEdges');
        expect(NOTIFICATIONS).toContain('AnalogLow');
        expect(NOTIFICATIONS).toContain('AnalogHigh');
        expect(NOTIFICATIONS).toContain('AnalogBoth');
        expect(NOTIFICATIONS).toHaveLength(7);
    });

    it('contains predefined color palette', () => {
        expect(PALETTE.length).toBeGreaterThanOrEqual(8);
        for (const color of PALETTE) {
            expect(color).toMatch(/^#[0-9A-Fa-f]{6}$/);
        }
    });
});
