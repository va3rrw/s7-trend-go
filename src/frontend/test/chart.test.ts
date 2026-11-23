import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TrendChart } from '../src/chart';
import { backend } from '../src/store';

describe('TrendChart', () => {
    let canvas: HTMLCanvasElement;
    let boolBand: HTMLElement;
    let timeAxis: HTMLElement;

    beforeEach(() => {
        document.body.innerHTML = `
            <canvas id="trendCanvas"></canvas>
            <div id="booleanBand"></div>
            <div id="timeAxis"></div>
        `;
        canvas = document.getElementById('trendCanvas') as HTMLCanvasElement;
        boolBand = document.getElementById('booleanBand') as HTMLElement;
        timeAxis = document.getElementById('timeAxis') as HTMLElement;
    });

    it('initializes cleanly without errors', () => {
        const onDrag = vi.fn();
        const trend = new TrendChart(canvas, boolBand, timeAxis, onDrag);
        expect(trend).toBeDefined();
        trend.destroy();
    });

    it('updates tags and axes', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tags = [
            {
                id: 'tag-1',
                name: 'Analog_1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            },
            {
                id: 'tag-2',
                name: 'Digital_1',
                color: '#00FF00',
                dataType: 'Bool',
                yAxis: 'Y-Axis 1',
                enabled: true,
            },
        ];
        const axes = [
            { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
        ];

        expect(() => trend.setTags(tags, axes)).not.toThrow();
        trend.destroy();
    });

    it('handles window resize and mode changes', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        expect(() => trend.setWindow(120)).not.toThrow();
        expect(() => trend.setInterpolation('Line')).not.toThrow();
        expect(() => trend.setInterpolation('Step')).not.toThrow();
        expect(() => trend.setInterpolation('Cubic')).not.toThrow();
        expect(() => trend.setPaused(true)).not.toThrow();
        expect(() => trend.setPaused(false)).not.toThrow();
        trend.destroy();
    });

    it('accepts data points and handles clear', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const now = new Date().toISOString();

        expect(() => {
            trend.addDataPoint('tag-1', now, 42.5);
            trend.addDataPoint('tag-1', now, 43.1);
            trend.addDataPoint('tag-1', now, NaN); // should be ignored
        }).not.toThrow();

        expect(() => trend.clear()).not.toThrow();
        trend.destroy();
    });

    it('renders with direct canvas elements for boolean band and time axis', () => {
        const boolCanvas = document.createElement('canvas');
        const timeCanvas = document.createElement('canvas');
        const trend = new TrendChart(canvas, boolCanvas, timeCanvas);

        const tags = [
            {
                id: 'bool-1',
                name: 'Motor_Running',
                color: '#86EFAC',
                dataType: 'Bool',
                yAxis: 'Y-Axis 1',
                enabled: true,
            },
        ];
        trend.setTags(tags, [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }]);

        const now = new Date().toISOString();
        trend.addDataPoint('bool-1', now, 1);

        expect(() => trend.clear()).not.toThrow();
        trend.destroy();
    });

    it('renders more than 8 boolean tags in the boolean band without 8-tag limit', async () => {
        const boolCanvas = document.createElement('canvas');
        const timeCanvas = document.createElement('canvas');
        const trend = new TrendChart(canvas, boolCanvas, timeCanvas);

        const tags = Array.from({ length: 16 }, (_, i) => ({
            id: `bool-${i}`,
            name: `Bit_${i}`,
            color: '#86EFAC',
            dataType: 'Bool',
            yAxis: 'Y-Axis 1',
            enabled: true,
        }));
        trend.setTags(tags, [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }]);

        await new Promise((resolve) => requestAnimationFrame(resolve));

        // 16 rows * 24px = 384px height
        expect(boolCanvas.style.height).toBe('384px');
        trend.destroy();
    });

    it('handles window resize events cleanly', () => {
        const boolCanvas = document.createElement('canvas');
        const timeCanvas = document.createElement('canvas');
        const trend = new TrendChart(canvas, boolCanvas, timeCanvas);

        expect(() => {
            window.dispatchEvent(new Event('resize'));
        }).not.toThrow();

        trend.destroy();
    });

    it('retains history points in rolling buffer during live streaming', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        trend.setWindow(60); // 60 seconds

        const baseTime = 1000000;
        // Add point at baseTime
        trend.addDataPoint('tag-1', new Date(baseTime).toISOString(), 10);
        // Add point at baseTime + 200 seconds
        trend.addDataPoint('tag-1', new Date(baseTime + 200000).toISOString(), 20);

        // Both points should be retained in memory for smooth review/panning
        const historyMap = (trend as any).history.get('tag-1');
        expect(historyMap.length).toBe(2);
        expect(historyMap[0].value).toBe(10);
        expect(historyMap[1].value).toBe(20);

        trend.destroy();
    });

    it('efficiently handles high-throughput point streams and caps at max buffer capacity', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        trend.setWindow(30);

        const baseTime = 1000000;
        // Stream 60,000 points
        for (let i = 0; i < 60000; i++) {
            trend.addDataPoint('tag-fast', new Date(baseTime + i * 10).toISOString(), i);
        }

        const pts = (trend as any).history.get('tag-fast');
        expect(pts.length).toBeGreaterThan(0);
        expect(pts.length).toBeLessThanOrEqual(55000);
        // Last point should match the last emitted value
        expect(pts[pts.length - 1].value).toBe(59999);

        trend.destroy();
    });

    it('enables cursors, calculates measurements, and emits cursor changes', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        trend.setTags(
            [
                {
                    id: 'tag-analog',
                    name: 'FlowRate',
                    color: '#38BDF8',
                    dataType: 'Real',
                    yAxis: 'Y-Axis 1',
                    enabled: true,
                },
            ],
            [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }],
        );

        let measurementResult: any = null;
        trend.setOnCursorChange((meas) => {
            measurementResult = meas;
        });

        trend.setCursorsEnabled(true);
        expect(trend.getMeasurements()).not.toBeNull();

        trend.addDataPoint('tag-analog', new Date(1000000).toISOString(), 25);
        trend.addDataPoint('tag-analog', new Date(1005000).toISOString(), 75);

        trend.fitCursorsToWindow();
        const meas = trend.getMeasurements();
        expect(meas).toBeDefined();
        expect(meas?.tags['tag-analog']).toBeDefined();

        trend.setCursorsEnabled(false);
        expect(trend.getMeasurements()).toBeNull();

        trend.destroy();
    });

    it('handles mouse wheel zoom in and zoom out centered on cursor and triggers pause', () => {
        const onWinChange = vi.fn();
        const onDragStart = vi.fn();
        const trend = new TrendChart(canvas, boolBand, timeAxis, onDragStart);
        trend.setOnWindowChange(onWinChange);
        trend.setWindow(60);

        // Dispatch wheel event scrolling UP (zoom in)
        const zoomInEvent = new WheelEvent('wheel', {
            deltaY: -100,
            clientX: 200,
            clientY: 100,
            bubbles: true,
            cancelable: true,
        });
        canvas.dispatchEvent(zoomInEvent);

        expect(onWinChange).toHaveBeenCalled();
        expect(onDragStart).toHaveBeenCalledTimes(1);
        const zoomedInWindow = onWinChange.mock.calls[0][0];
        expect(zoomedInWindow).toBeLessThan(60);

        // Reset and verify unpausing resets the window back to original configured 60s
        trend.setPaused(false);
        expect(onWinChange).toHaveBeenCalledWith(60);
        onDragStart.mockClear();

        // Dispatch wheel event scrolling DOWN (zoom out)
        const zoomOutEvent = new WheelEvent('wheel', {
            deltaY: 100,
            clientX: 200,
            clientY: 100,
            bubbles: true,
            cancelable: true,
        });
        canvas.dispatchEvent(zoomOutEvent);
        expect(onDragStart).toHaveBeenCalledTimes(1);
        const zoomedOutWindow = onWinChange.mock.calls[onWinChange.mock.calls.length - 1][0];
        expect(zoomedOutWindow).toBeGreaterThan(60);

        // Resume after zoom out and verify it resets back to 60s
        trend.setPaused(false);
        expect(onWinChange.mock.calls[onWinChange.mock.calls.length - 1][0]).toBe(60);

        trend.destroy();
    });

    it('pauses live chart and triggers drag start callback when cursors are enabled, and deactivates cursors on resume', () => {
        const onDragStart = vi.fn();
        const onCursorChange = vi.fn();
        const trend = new TrendChart(canvas, boolBand, timeAxis, onDragStart);
        trend.setOnCursorChange(onCursorChange);

        expect((trend as any).paused).toBe(false);

        // Enable cursors while chart is live
        trend.setCursorsEnabled(true);

        expect((trend as any).paused).toBe(true);
        expect((trend as any).pausedAnchorTime).not.toBeNull();
        expect((trend as any).cursorsEnabled).toBe(true);
        expect(onDragStart).toHaveBeenCalledTimes(1);

        // Resuming chart should deactivate cursors
        trend.setPaused(false);
        expect((trend as any).paused).toBe(false);
        expect((trend as any).cursorsEnabled).toBe(false);
        expect(trend.getMeasurements()).toBeNull();

        trend.destroy();
    });

    it('clears ruler times when resuming the chart', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);

        trend.setCursorsEnabled(true);
        expect((trend as any).cursorA).not.toBeNull();
        expect((trend as any).cursorB).not.toBeNull();

        trend.setPaused(false);

        expect((trend as any).cursorsEnabled).toBe(false);
        expect((trend as any).cursorA).toBeNull();
        expect((trend as any).cursorB).toBeNull();

        trend.destroy();
    });

    it('handles large datasets without call stack overflow in getAxisRange (regression)', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tags = Array.from({ length: 12 }, (_, i) => ({
            id: `tag-${i}`,
            name: `Tag_${i}`,
            color: '#38BDF8',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        }));
        const axis = { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true };
        trend.setTags(tags, [axis]);

        const baseTime = 1000000;
        // 12 tags x 15,000 points = 180,000 values (> 125,000 V8 call stack spread limit)
        for (let j = 0; j < 15000; j++) {
            const timeStr = new Date(baseTime + j * 200).toISOString();
            for (let i = 0; i < 12; i++) {
                // Ensure data exceeds default 0..100 bounds on both sides
                trend.addDataPoint(`tag-${i}`, timeStr, (j % 50) * 3 - 25 + i);
            }
        }

        let range: { min: number; max: number } | undefined;
        expect(() => {
            range = (trend as any).getAxisRange(axis, tags);
        }).not.toThrow();

        expect(range).toBeDefined();
        // Min value is -25, max value is 49*3 - 25 + 11 = 133
        // Both exceed 0..100, so range should cover the full data
        expect(range!.min).toBeLessThanOrEqual(-25);
        expect(range!.max).toBeGreaterThanOrEqual(133);

        trend.destroy();
    });

    it('applies padding and expansion semantics correctly in getAxisRange', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tags = [
            {
                id: 'tag-1',
                name: 'Analog_1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            },
        ];
        const axis = { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true };

        // 1. With no points, returns initial range without modification
        const emptyRange = (trend as any).getAxisRange(axis, tags);
        expect(emptyRange).toEqual({ min: 0, max: 100 });

        // 2. If range.max <= range.min, resets to { min: 0, max: 100 }
        const invalidAxis = { name: 'Y-Axis 2', minimum: 50, maximum: 20, autoScale: true };
        const resetRange = (trend as any).getAxisRange(invalidAxis, []);
        expect(resetRange).toEqual({ min: 0, max: 100 });

        // 3. Data within range does not expand range
        trend.addDataPoint('tag-1', new Date(1000).toISOString(), 50);
        const insideRange = (trend as any).getAxisRange(axis, tags);
        expect(insideRange).toEqual({ min: 0, max: 100 });

        // 4. Data exceeding range expands with padding = Math.max(0.001, (max-min)*0.08)
        // Data range: [-50, 150], diff = 200, pad = 200 * 0.08 = 16
        trend.addDataPoint('tag-1', new Date(2000).toISOString(), -50);
        trend.addDataPoint('tag-1', new Date(3000).toISOString(), 150);
        const expandedRange = (trend as any).getAxisRange(axis, tags);
        expect(expandedRange.min).toBeCloseTo(-50 - 16, 5);
        expect(expandedRange.max).toBeCloseTo(150 + 16, 5);

        // 5. Non-finite values in history do not corrupt range calculation
        (trend as any).history.get('tag-1').push({ timestamp: 4000, value: NaN });
        (trend as any).history.get('tag-1').push({ timestamp: 5000, value: Infinity });
        const safeRange = (trend as any).getAxisRange(axis, tags);
        expect(Number.isFinite(safeRange.min)).toBe(true);
        expect(Number.isFinite(safeRange.max)).toBe(true);

        trend.destroy();
    });

    it('catches render errors inside requestAnimationFrame callback and keeps render loop alive', async () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

        // Force doRender to throw on first call
        let throwOnce = true;
        vi.spyOn(trend as any, 'doRender').mockImplementation(() => {
            if (throwOnce) {
                throwOnce = false;
                throw new Error('Simulated render explosion');
            }
        });

        // Trigger render
        (trend as any).render();

        // Wait for rAF callback to execute
        await new Promise((resolve) => requestAnimationFrame(resolve));

        expect(consoleErrorSpy).toHaveBeenCalledWith(expect.any(Error));
        // Verify renderPending is reset to false so subsequent renders can proceed
        expect((trend as any).renderPending).toBe(false);

        // Next render should succeed without throwing
        expect(() => (trend as any).render()).not.toThrow();
        await new Promise((resolve) => requestAnimationFrame(resolve));

        consoleErrorSpy.mockRestore();
        trend.destroy();
    });

    it('does not re-query GetHistoryRange after a fetch resolves for a range', async () => {
        const getHistoryRangeMock = vi.fn().mockResolvedValue({
            'tag-1': [
                { t: 900000, v: 10 },
                { t: 950000, v: 20 },
            ],
        });
        backend()!.GetHistoryRange = getHistoryRangeMock;

        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tags = [
            {
                id: 'tag-1',
                name: 'T1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            },
        ];
        trend.setTags(tags, [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }]);

        // Add a live point at 1,000,000
        trend.addDataPoint('tag-1', new Date(1000000).toISOString(), 50);

        // View extends before earliest live point -> needs history
        (trend as any).viewOffsetSeconds = -10;
        (trend as any).doRender();

        await vi.waitFor(() => {
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(1);
        });

        // Re-render the same view range
        (trend as any).doRender();
        await new Promise((resolve) => setTimeout(resolve, 60));

        // GetHistoryRange must NOT be called again
        expect(getHistoryRangeMock).toHaveBeenCalledTimes(1);

        trend.destroy();
    });

    it('zooms to 86400s with bounded backend data without throwing or freezing', async () => {
        const mockData: Record<string, Array<{ t: number; v: number }>> = {};
        for (let i = 0; i < 12; i++) {
            mockData[`tag-${i}`] = Array.from({ length: 1000 }, (_, idx) => ({
                t: 1000000 + idx * 80000,
                v: idx % 100,
            }));
        }
        backend()!.GetHistoryRange = vi.fn().mockResolvedValue(mockData);

        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tags = Array.from({ length: 12 }, (_, i) => ({
            id: `tag-${i}`,
            name: `Tag_${i}`,
            color: '#38BDF8',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        }));
        trend.setTags(tags, [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }]);

        expect(() => {
            trend.setWindow(86400);
        }).not.toThrow();

        await vi.waitFor(() => {
            expect((trend as any).fetchedHistory).not.toBeNull();
        });

        expect(() => (trend as any).doRender()).not.toThrow();
        trend.setCursorsEnabled(true);
        expect(() => trend.getMeasurements()).not.toThrow();

        trend.destroy();
    });

    it('does not inflate the live history buffer with fetched data', async () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tags = [
            {
                id: 'tag-1',
                name: 'T1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            },
        ];
        trend.setTags(tags, [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }]);

        // Add 3 live points
        trend.addDataPoint('tag-1', new Date(1000000).toISOString(), 10);
        trend.addDataPoint('tag-1', new Date(1001000).toISOString(), 20);
        trend.addDataPoint('tag-1', new Date(1002000).toISOString(), 30);

        const liveBefore = (trend as any).history.get('tag-1');
        expect(liveBefore.length).toBe(3);

        // Mock 500 older historical points
        const olderPoints = Array.from({ length: 500 }, (_, idx) => ({
            t: 500000 + idx * 1000,
            v: idx,
        }));
        backend()!.GetHistoryRange = vi.fn().mockResolvedValue({
            'tag-1': olderPoints,
        });

        // Trigger history fetch by panning back
        (trend as any).viewOffsetSeconds = -100;
        (trend as any).doRender();

        await vi.waitFor(() => {
            expect((trend as any).fetchedHistory).not.toBeNull();
        });

        // Live buffer must remain at exactly 3 points
        const liveAfter = (trend as any).history.get('tag-1');
        expect(liveAfter.length).toBe(3);
        expect(liveAfter[0].timestamp).toBe(1000000);
        expect(liveAfter[2].timestamp).toBe(1002000);

        // Combined points via pointsForTag includes both: 500 fetched + 3 live = 503 points
        const combined = trend.pointsForTag('tag-1');
        expect(combined.length).toBe(503);

        trend.destroy();
    });

    it('preserves history when disabling an analog tag and restores its line on re-enable', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tag1 = {
            id: 'tag-1',
            name: 'Analog_1',
            color: '#FF0000',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        };
        const tag2 = {
            id: 'tag-2',
            name: 'Analog_2',
            color: '#00FF00',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        };
        const axes = [
            { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
        ];

        trend.setTags([tag1, tag2], axes);
        trend.addDataPoint('tag-1', new Date(1000000).toISOString(), 10);
        trend.addDataPoint('tag-2', new Date(1000000).toISOString(), 20);
        trend.addDataPoint('tag-1', new Date(1001000).toISOString(), 11);
        trend.addDataPoint('tag-2', new Date(1001000).toISOString(), 21);
        trend.addDataPoint('tag-1', new Date(1002000).toISOString(), 12);
        trend.addDataPoint('tag-2', new Date(1002000).toISOString(), 22);

        (trend as any).doRender();
        expect((trend as any).chart.data.datasets.length).toBe(2);

        // Disable tag-1; its line disappears, but its history is preserved
        trend.setTags([{ ...tag1, enabled: false }, tag2], axes);
        (trend as any).doRender();
        expect((trend as any).chart.data.datasets.map((d: any) => d.label)).toEqual([
            'Analog_2',
        ]);
        expect((trend as any).history.get('tag-1')?.length).toBe(3);

        // New points arrive for tag-2 while tag-1 is disabled
        trend.addDataPoint('tag-2', new Date(1003000).toISOString(), 23);
        trend.addDataPoint('tag-2', new Date(1004000).toISOString(), 24);

        // Re-enable tag-1 and plot a new point for it
        trend.setTags([{ ...tag1, enabled: true }, tag2], axes);
        trend.addDataPoint('tag-1', new Date(1005000).toISOString(), 15);
        (trend as any).doRender();

        const datasets = (trend as any).chart.data.datasets;
        expect(datasets.map((d: any) => d.label)).toEqual(['Analog_1', 'Analog_2']);
        expect(datasets[0].data).toEqual([
            { x: 1000000, y: 10 },
            { x: 1001000, y: 11 },
            { x: 1002000, y: 12 },
            { x: 1005000, y: 15 },
        ]);

        trend.destroy();
    });

    it('deletes history when a tag is removed from tags list', () => {
        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tag1 = {
            id: 'tag-1',
            name: 'Analog_1',
            color: '#FF0000',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        };
        const tag2 = {
            id: 'tag-2',
            name: 'Analog_2',
            color: '#00FF00',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        };
        const axes = [
            { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
        ];

        trend.setTags([tag1, tag2], axes);
        trend.addDataPoint('tag-1', new Date(1000000).toISOString(), 10);
        trend.addDataPoint('tag-2', new Date(1000000).toISOString(), 20);

        // Removing tag-1 entirely (not just disabling) should prune its history
        trend.setTags([tag2], axes);
        expect((trend as any).history.has('tag-1')).toBe(false);
        expect((trend as any).history.has('tag-2')).toBe(true);

        trend.destroy();
    });

    it('fetches backend history for a newly enabled tag even when another enabled tag covers window start', async () => {
        const getHistoryRangeMock = vi.fn().mockResolvedValue({
            'tag-1': [
                { t: 985000, v: 31 },
                { t: 990000, v: 32 },
            ],
            'tag-2': [],
        });
        backend()!.GetHistoryRange = getHistoryRangeMock;

        const trend = new TrendChart(canvas, boolBand, timeAxis);
        const tag1 = {
            id: 'tag-1',
            name: 'Analog_1',
            color: '#FF0000',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: false,
        };
        const tag2 = {
            id: 'tag-2',
            name: 'Analog_2',
            color: '#00FF00',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            enabled: true,
        };
        const axes = [
            { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
        ];

        trend.setTags([tag1, tag2], axes);
        // tag-2 covers [940000..1010000], so its earliest point precedes window start (950000)
        trend.addDataPoint('tag-2', new Date(940000).toISOString(), 5);
        trend.addDataPoint('tag-2', new Date(1010000).toISOString(), 6);

        // Enable tag-1 (no in-memory points yet) and stream one new live point
        trend.setTags([{ ...tag1, enabled: true }, tag2], axes);
        trend.addDataPoint('tag-1', new Date(1010500).toISOString(), 35);
        (trend as any).doRender();

        await vi.waitFor(() => {
            expect(getHistoryRangeMock).toHaveBeenCalled();
            expect((trend as any).fetchedHistory).not.toBeNull();
        });

        (trend as any).doRender();
        const analog1Dataset = (trend as any).chart.data.datasets.find(
            (d: any) => d.label === 'Analog_1',
        );
        expect(analog1Dataset.data).toEqual([
            { x: 985000, y: 31 },
            { x: 990000, y: 32 },
            { x: 1010500, y: 35 },
        ]);

        trend.destroy();
    });

    it('does not starve a scheduled GetHistoryRange fetch during fast live updates', async () => {
        vi.useFakeTimers();
        try {
            const getHistoryRangeMock = vi.fn().mockResolvedValue({
                'tag-1': [{ t: 970000, v: 42 }],
                'tag-2': [],
            });
            backend()!.GetHistoryRange = getHistoryRangeMock;

            const trend = new TrendChart(canvas, boolBand, timeAxis);
            const tag1 = {
                id: 'tag-1',
                name: 'Analog_1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            };
            const tag2 = {
                id: 'tag-2',
                name: 'Analog_2',
                color: '#00FF00',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            };
            const axes = [
                { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
            ];

            trend.setTags([tag1, tag2], axes);
            trend.addDataPoint('tag-2', new Date(900000).toISOString(), 1);
            trend.addDataPoint('tag-2', new Date(1000000).toISOString(), 2);

            (trend as any).doRender();
            expect(getHistoryRangeMock).not.toHaveBeenCalled();

            // Simulate fast live updates every 20ms (< 40ms debounce)
            await vi.advanceTimersByTimeAsync(20);
            trend.addDataPoint('tag-2', new Date(1000020).toISOString(), 3);
            (trend as any).doRender();

            await vi.advanceTimersByTimeAsync(25);
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(1);

            trend.destroy();
        } finally {
            vi.useRealTimers();
        }
    });

    it('ignores stale in-flight GetHistoryRange response after enabled tags change and refetches', async () => {
        vi.useFakeTimers();
        try {
            let resolveFirst!: (val: Record<string, Array<{ t: number; v: number }>>) => void;
            const firstPromise = new Promise<Record<string, Array<{ t: number; v: number }>>>(
                (resolve) => {
                    resolveFirst = resolve;
                },
            );

            const getHistoryRangeMock = vi
                .fn()
                .mockReturnValueOnce(firstPromise)
                .mockResolvedValueOnce({
                    'tag-1': [{ t: 980000, v: 11 }],
                    'tag-2': [{ t: 990000, v: 88 }],
                });
            backend()!.GetHistoryRange = getHistoryRangeMock;

            const trend = new TrendChart(canvas, boolBand, timeAxis);
            const tag1 = {
                id: 'tag-1',
                name: 'Analog_1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            };
            const tag2 = {
                id: 'tag-2',
                name: 'Analog_2',
                color: '#00FF00',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: false,
            };
            const axes = [
                { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
            ];

            trend.setTags([tag1, tag2], axes);
            trend.addDataPoint('tag-1', new Date(1000000).toISOString(), 10);
            (trend as any).doRender();

            // Advance past debounce to start first in-flight fetch (for tag-1 only)
            await vi.advanceTimersByTimeAsync(50);
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(1);
            expect(getHistoryRangeMock.mock.calls[0][0]).toEqual(['tag-1']);

            // While first request is still in-flight, enable tag-2
            trend.setTags([tag1, { ...tag2, enabled: true }], axes);

            // Resolve stale first response; must be ignored and trigger fresh fetch
            resolveFirst({ 'tag-1': [{ t: 980000, v: 11 }] });
            await Promise.resolve();
            await Promise.resolve();

            // Advance rAF + debounce for the retry fetch
            await vi.advanceTimersByTimeAsync(100);

            expect(getHistoryRangeMock).toHaveBeenCalledTimes(2);
            expect(getHistoryRangeMock.mock.calls[1][0]).toEqual(['tag-1', 'tag-2']);
            expect((trend as any).fetchedHistory?.data.get('tag-2')).toEqual([
                { timestamp: 990000, value: 88 },
            ]);

            trend.destroy();
        } finally {
            vi.useRealTimers();
        }
    });

    it('cancels superseded in-flight cancellable GetHistoryRange requests on pan, setTags, clear, and destroy', async () => {
        vi.useFakeTimers();
        try {
            const makeCancellable = () => {
                let resolve!: (val: Record<string, Array<{ t: number; v: number }>>) => void;
                let reject!: (err: any) => void;
                const cancel = vi.fn(() => {
                    reject(new Error('CancelError'));
                });
                const promise: any = new Promise((res, rej) => {
                    resolve = res;
                    reject = rej;
                });
                promise.cancel = cancel;
                return { promise, resolve, cancel };
            };

            const req1 = makeCancellable();
            const req2 = makeCancellable();
            const req3 = makeCancellable();
            const req4 = makeCancellable();

            const getHistoryRangeMock = vi
                .fn()
                .mockReturnValueOnce(req1.promise)
                .mockReturnValueOnce(req2.promise)
                .mockReturnValueOnce(req3.promise)
                .mockReturnValueOnce(req4.promise);
            backend()!.GetHistoryRange = getHistoryRangeMock;

            const trend = new TrendChart(canvas, boolBand, timeAxis);
            const tag1 = {
                id: 'tag-1',
                name: 'Analog_1',
                color: '#FF0000',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: true,
            };
            const tag2 = {
                id: 'tag-2',
                name: 'Analog_2',
                color: '#00FF00',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                enabled: false,
            };
            const axes = [
                { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
            ];

            trend.setTags([tag1, tag2], axes);
            trend.addDataPoint('tag-1', new Date(1000000).toISOString(), 10);
            (trend as any).viewOffsetSeconds = -50;
            (trend as any).doRender();

            await vi.advanceTimersByTimeAsync(50);
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(1);

            // 1. Pan further back while req1 is still pending -> cancels req1 and starts req2
            (trend as any).viewOffsetSeconds = -250;
            (trend as any).doRender();
            expect(req1.cancel).toHaveBeenCalledWith('superseded');

            await vi.advanceTimersByTimeAsync(50);
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(2);

            // 2. Enable tag-2 while req2 is in flight -> cancels req2 and starts req3
            trend.setTags([tag1, { ...tag2, enabled: true }], axes);
            expect(req2.cancel).toHaveBeenCalledWith('superseded');

            await vi.advanceTimersByTimeAsync(100);
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(3);

            // 3. Clear while req3 is in flight -> cancels req3
            trend.clear();
            expect(req3.cancel).toHaveBeenCalledWith('superseded');

            // 4. Trigger req4 then destroy while in flight -> cancels req4
            (trend as any).viewOffsetSeconds = -100;
            (trend as any).doRender();
            await vi.advanceTimersByTimeAsync(50);
            expect(getHistoryRangeMock).toHaveBeenCalledTimes(4);

            trend.destroy();
            expect(req4.cancel).toHaveBeenCalledWith('superseded');
        } finally {
            vi.useRealTimers();
        }
    });
});


