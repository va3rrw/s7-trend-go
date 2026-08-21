import { describe, it, expect } from 'vitest';
import {
    niceStepAtMost,
    chooseXGridIntervals,
    formatTickTime,
    isMajorTick,
    mergePoints,
    computePlotLayout,
} from '../src/canvasHelpers';

describe('canvasHelpers', () => {
    it('calculates niceStepAtMost correctly', () => {
        expect(niceStepAtMost(0.5)).toBe(1);
        expect(niceStepAtMost(10)).toBe(10);
        expect(niceStepAtMost(25)).toBe(15);
        expect(niceStepAtMost(65)).toBe(60);
        expect(niceStepAtMost(5000)).toBe(3600);
    });

    it('chooses reasonable X grid intervals for different time spans', () => {
        const sixtySec = chooseXGridIntervals(60);
        expect(sixtySec.majorMs).toBe(5000);
        expect(sixtySec.stepMs).toBe(500);

        const fiveMin = chooseXGridIntervals(300);
        expect(fiveMin.majorMs).toBe(30000);
        expect(fiveMin.stepMs).toBe(3000);
    });

    it('formats tick time with leading zeros', () => {
        const d = new Date(2026, 0, 1, 9, 5, 7);
        expect(formatTickTime(d.getTime())).toBe('09:05:07');
    });

    it('identifies major ticks correctly', () => {
        const majorMs = 10000;
        const stepMs = 1000;
        expect(isMajorTick(20000, majorMs, stepMs)).toBe(true);
        expect(isMajorTick(21000, majorMs, stepMs)).toBe(false);
        expect(isMajorTick(30000, majorMs, stepMs)).toBe(true);
    });

    it('merges point buffers correctly within capacity', () => {
        const existing = [
            { timestamp: 1000, value: 10 },
            { timestamp: 2000, value: 20 },
        ];
        const fetched = [
            { timestamp: 500, value: 5 },
            { timestamp: 1000, value: 15 }, // overwrite existing timestamp
            { timestamp: 3000, value: 30 },
        ];

        const merged = mergePoints(existing, fetched, 5);
        expect(merged).toEqual([
            { timestamp: 500, value: 5 },
            { timestamp: 1000, value: 15 },
            { timestamp: 2000, value: 20 },
            { timestamp: 3000, value: 30 },
        ]);
    });

    it('computes plot layout geometry', () => {
        const canvas = document.createElement('canvas');
        Object.defineProperty(canvas, 'clientWidth', { value: 800 });

        const layout = computePlotLayout(canvas, {
            left: 40,
            right: 760,
            width: 720,
        });

        expect(layout.left).toBe(40);
        expect(layout.right).toBe(40);
        expect(layout.width).toBe(720);
        expect(layout.canvasW).toBe(800);
    });
});
