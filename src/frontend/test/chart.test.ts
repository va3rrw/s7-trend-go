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
        expect(() => trend.setInterpolation('Differential')).not.toThrow();
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

    it('handles window resize events cleanly', () => {
        const boolCanvas = document.createElement('canvas');
        const timeCanvas = document.createElement('canvas');
        const trend = new TrendChart(canvas, boolCanvas, timeCanvas);

        expect(() => {
            window.dispatchEvent(new Event('resize'));
        }).not.toThrow();

        trend.destroy();
    });
});

