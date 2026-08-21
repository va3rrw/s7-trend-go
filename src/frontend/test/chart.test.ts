import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TrendChart } from '../src/chart';

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

    it('handles mouse wheel zoom in and zoom out centered on cursor', () => {
        const onWinChange = vi.fn();
        const trend = new TrendChart(canvas, boolBand, timeAxis);
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
        const zoomedWindow = onWinChange.mock.calls[0][0];
        expect(zoomedWindow).toBeLessThan(60);

        // Dispatch wheel event scrolling DOWN (zoom out)
        const zoomOutEvent = new WheelEvent('wheel', {
            deltaY: 100,
            clientX: 200,
            clientY: 100,
            bubbles: true,
            cancelable: true,
        });
        canvas.dispatchEvent(zoomOutEvent);
        expect(onWinChange.mock.calls.length).toBe(2);

        trend.destroy();
    });
});



