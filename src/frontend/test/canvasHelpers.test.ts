import { describe, it, expect, vi } from 'vitest';
import {
    niceStepAtMost,
    chooseXGridIntervals,
    formatTickTime,
    isMajorTick,
    mergePoints,
    computePlotLayout,
    drawBooleanBand,
} from '../src/canvasHelpers';
import { CHART_LAYOUT } from '../src/chartConstants';

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

    describe('drawBooleanBand', () => {
        function createMockCtx() {
            return {
                clearRect: vi.fn(),
                fillRect: vi.fn(),
                stroke: vi.fn(),
                beginPath: vi.fn(),
                moveTo: vi.fn(),
                lineTo: vi.fn(),
                setLineDash: vi.fn(),
                save: vi.fn(),
                restore: vi.fn(),
                rect: vi.fn(),
                clip: vi.fn(),
                fillText: vi.fn(),
            } as unknown as CanvasRenderingContext2D;
        }

        const baseTag = {
            id: 'bool-1',
            name: 'Motor_On',
            color: '#86EFAC',
            dataType: 'Bool',
            yAxis: '',
            enabled: true,
        };

        const layout = {
            left: 100,
            right: 0,
            width: 600,
            canvasW: 700,
        };

        it('merges consecutive HIGH samples into a single contiguous fillRect bar', () => {
            const ctx = createMockCtx();
            const points = Array.from({ length: 20 }, (_, i) => ({
                timestamp: 1000 + i * 100,
                value: 1,
            }));

            drawBooleanBand(ctx, {
                start: 0,
                end: 10000,
                layout,
                boolTags: [baseTag],
                history: new Map([[baseTag.id, points]]),
                cursorsEnabled: false,
                cursorA: null,
                cursorB: null,
            });

            const barCalls = (ctx.fillRect as ReturnType<typeof vi.fn>).mock.calls.filter(
                (args) => args[1] === CHART_LAYOUT.boolBarPaddingY,
            );
            expect(barCalls).toHaveLength(1);
            // 1000..10000 out of 0..10000 -> x1 = 100 + 0.1*600 = 160, x2 = 700
            expect(barCalls[0]).toEqual([
                160,
                CHART_LAYOUT.boolBarPaddingY,
                540,
                CHART_LAYOUT.boolBarHeight,
            ]);
        });

        it('splits separate HIGH runs around LOW samples', () => {
            const ctx = createMockCtx();
            const points = [
                { timestamp: 1000, value: 1 },
                { timestamp: 1500, value: 1 },
                { timestamp: 2000, value: 0 },
                { timestamp: 3000, value: 0 },
                { timestamp: 5000, value: 1 },
                { timestamp: 6000, value: 1 },
                { timestamp: 7000, value: 0 },
            ];

            drawBooleanBand(ctx, {
                start: 0,
                end: 10000,
                layout,
                boolTags: [baseTag],
                history: new Map([[baseTag.id, points]]),
                cursorsEnabled: false,
                cursorA: null,
                cursorB: null,
            });

            const barCalls = (ctx.fillRect as ReturnType<typeof vi.fn>).mock.calls.filter(
                (args) => args[1] === CHART_LAYOUT.boolBarPaddingY,
            );
            expect(barCalls).toHaveLength(2);
            // Run 1: 1000..2000 -> x = 160..220 (w=60)
            expect(barCalls[0]).toEqual([
                160,
                CHART_LAYOUT.boolBarPaddingY,
                60,
                CHART_LAYOUT.boolBarHeight,
            ]);
            // Run 2: 5000..7000 -> x = 400..520 (w=120)
            expect(barCalls[1]).toEqual([
                400,
                CHART_LAYOUT.boolBarPaddingY,
                120,
                CHART_LAYOUT.boolBarHeight,
            ]);
        });

        it('clamps HIGH runs starting before start and continuing past end cleanly', () => {
            const ctx = createMockCtx();
            const points = [{ timestamp: 500, value: 1 }];

            drawBooleanBand(ctx, {
                start: 1000,
                end: 5000,
                layout,
                boolTags: [baseTag],
                history: new Map([[baseTag.id, points]]),
                cursorsEnabled: false,
                cursorA: null,
                cursorB: null,
            });

            const barCalls = (ctx.fillRect as ReturnType<typeof vi.fn>).mock.calls.filter(
                (args) => args[1] === CHART_LAYOUT.boolBarPaddingY,
            );
            expect(barCalls).toHaveLength(1);
            expect(barCalls[0]).toEqual([
                100,
                CHART_LAYOUT.boolBarPaddingY,
                600,
                CHART_LAYOUT.boolBarHeight,
            ]);
        });
    });
});
