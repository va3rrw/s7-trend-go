import { describe, it, expect } from 'vitest';
import {
    interpolateValue,
    calculateTagStats,
    type Point,
} from '../src/measurement';

describe('measurement.ts', () => {
    const samplePoints: Point[] = [
        { timestamp: 1000, value: 10 },
        { timestamp: 2000, value: 20 },
        { timestamp: 3000, value: 40 },
        { timestamp: 4000, value: 30 },
        { timestamp: 5000, value: 50 },
    ];

    it('interpolates values correctly', () => {
        expect(interpolateValue([], 1000)).toBeNull();
        expect(interpolateValue(samplePoints, 500)).toBe(10); // before range
        expect(interpolateValue(samplePoints, 6000)).toBe(50); // after range
        expect(interpolateValue(samplePoints, 1000)).toBe(10); // exact match
        expect(interpolateValue(samplePoints, 1500)).toBe(15); // midpoint interpolation
        expect(interpolateValue(samplePoints, 2500)).toBe(30); // linear between 20 and 40
        expect(interpolateValue(samplePoints, 3500)).toBe(35); // linear between 40 and 30
    });

    it('calculates tag statistics and deltas between cursors', () => {
        // Cursor A at 1500 (valA = 15), Cursor B at 3500 (valB = 35)
        const stats = calculateTagStats(samplePoints, 1500, 3500);

        expect(stats.valA).toBe(15);
        expect(stats.valB).toBe(35);
        expect(stats.deltaY).toBe(20);
        // deltaT = (3500 - 1500) / 1000 = 2.0 s -> slope = 20 / 2 = 10 units/s
        expect(stats.slope).toBe(10);

        // Points included in range [1500, 3500]: 2000 (20), 3000 (40), plus interpolated 15 and 35
        expect(stats.min).toBe(15);
        expect(stats.max).toBe(40);
        expect(stats.mean).toBe((20 + 40 + 15 + 35) / 4); // 27.5
        expect(stats.stdDev).toBeGreaterThan(0);
    });

    it('handles reversed cursors (Cursor B before Cursor A)', () => {
        // Cursor A at 4000 (valA = 30), Cursor B at 2000 (valB = 20)
        const stats = calculateTagStats(samplePoints, 4000, 2000);

        expect(stats.valA).toBe(30);
        expect(stats.valB).toBe(20);
        expect(stats.deltaY).toBe(-10);
        // deltaT = (2000 - 4000) / 1000 = -2.0 s -> slope = -10 / -2.0 = 5 units/s
        expect(stats.slope).toBe(5);
        expect(stats.min).toBe(20);
        expect(stats.max).toBe(40);
    });
});
