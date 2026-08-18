import { describe, it, expect } from 'vitest';
import {
    newId,
    defaultSettings,
    DATA_TYPES,
    PALETTE,
    inferDataType,
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

    it('contains predefined color palette', () => {
        expect(PALETTE.length).toBeGreaterThanOrEqual(8);
        for (const color of PALETTE) {
            expect(color).toMatch(/^#[0-9A-Fa-f]{6}$/);
        }
    });

    it('infers data type based on address pattern', () => {
        // Bool
        expect(inferDataType('DB1.DBX0.0')).toBe('Bool');
        expect(inferDataType('db10.dbx4.7')).toBe('Bool');
        expect(inferDataType('M0.0')).toBe('Bool');
        expect(inferDataType('m10.5')).toBe('Bool');
        expect(inferDataType('I31.1')).toBe('Bool');
        expect(inferDataType('Q0.7')).toBe('Bool');
        expect(inferDataType('MX0.1')).toBe('Bool');
        expect(inferDataType('IX0.5')).toBe('Bool');
        expect(inferDataType('QX2.0')).toBe('Bool');

        // Byte
        expect(inferDataType('DB1.DBB0')).toBe('Byte');
        expect(inferDataType('MB10')).toBe('Byte');
        expect(inferDataType('IB0')).toBe('Byte');
        expect(inferDataType('QB4')).toBe('Byte');

        // Word / Int
        expect(inferDataType('DB1.DBW0')).toBe('Int');
        expect(inferDataType('DB1.DBW0', 'Word')).toBe('Word');
        expect(inferDataType('MW10')).toBe('Int');
        expect(inferDataType('IW64')).toBe('Int');
        expect(inferDataType('QW80')).toBe('Int');

        // Double Word / Real
        expect(inferDataType('DB1.DBD0')).toBe('Real');
        expect(inferDataType('DB1.DBD0', 'DInt')).toBe('DInt');
        expect(inferDataType('DB1.DBD0', 'DWord')).toBe('DWord');
        expect(inferDataType('MD20')).toBe('Real');
        expect(inferDataType('ID0')).toBe('Real');
        expect(inferDataType('QD80')).toBe('Real');

        // Empty or unrecognized
        expect(inferDataType('')).toBeNull();
        expect(inferDataType('invalid')).toBeNull();
    });
});

