import Chart from 'chart.js/auto';
import {
    interpolateValue,
    calculateTagStats,
    type Point,
    type CursorMeasurement,
    type TagStats,
} from './measurement';

export interface ChartTag {
    id: string;
    name: string;
    color: string;
    dataType: string;
    yAxis: string;
    enabled?: boolean;
}

export interface ChartAxis {
    name: string;
    minimum: number;
    maximum: number;
    autoScale: boolean;
}

export type { Point, CursorMeasurement, TagStats };

/** Nice time steps (seconds) used for major grid. */
const NICE_STEPS_SEC = [
    1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600,
];

/** Largest nice step that is still ≤ rawSeconds (prefer denser grids). */
function niceStepAtMost(rawSeconds: number): number {
    let best = NICE_STEPS_SEC[0];
    for (const s of NICE_STEPS_SEC) {
        if (s <= rawSeconds) best = s;
        else break;
    }
    return best;
}

/**
 * Pick minor + major grid steps from the visible span.
 * - Major (solid): about 10–12 lines across the window
 * - Minor (dashed): exactly 1/10 of major
 */
function chooseXGridIntervals(spanSeconds: number): {
    stepMs: number;
    majorMs: number;
} {
    const span = Math.max(1, spanSeconds);
    let majorSec = niceStepAtMost(span / 12);
    if (span / majorSec < 6 && majorSec > 1) {
        const idx = NICE_STEPS_SEC.indexOf(majorSec);
        if (idx > 0) majorSec = NICE_STEPS_SEC[idx - 1];
    }
    while (span / majorSec > 15) {
        const idx = NICE_STEPS_SEC.indexOf(majorSec);
        if (idx < 0 || idx >= NICE_STEPS_SEC.length - 1) break;
        majorSec = NICE_STEPS_SEC[idx + 1];
    }
    const minorSec = majorSec / 10;
    return { stepMs: minorSec * 1000, majorMs: majorSec * 1000 };
}

/** Fixed-width HH:mm:ss so label width never resizes the plot. */
function formatTickTime(ms: number): string {
    const d = new Date(ms);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function isMajorTick(
    timestamp: number,
    majorMs: number,
    stepMs: number,
): boolean {
    const rem = ((timestamp % majorMs) + majorMs) % majorMs;
    return rem < stepMs * 0.5 || rem > majorMs - stepMs * 0.5;
}

const secondGridPlugin = {
    id: 'secondGrid',
    beforeDraw(chart: any) {
        const scale = chart.scales.x;
        const area = chart.chartArea;
        if (!scale || !area) return;

        const min = Number(scale.min);
        const max = Number(scale.max);
        const spanSeconds = Math.max(1, (max - min) / 1000);
        const { stepMs, majorMs } = chooseXGridIntervals(spanSeconds);
        const first = Math.ceil(min / stepMs) * stepMs;
        const context = chart.ctx as CanvasRenderingContext2D;
        context.save();
        for (let timestamp = first; timestamp <= max; timestamp += stepMs) {
            const x = scale.getPixelForValue(timestamp);
            if (x < area.left || x > area.right) continue;
            const major = isMajorTick(timestamp, majorMs, stepMs);
            context.beginPath();
            context.setLineDash(major ? [] : [4, 4]);
            context.strokeStyle = major ? '#5B7182' : '#31414E';
            context.lineWidth = major ? 1 : 0.8;
            context.moveTo(x, area.top);
            context.lineTo(x, area.bottom);
            context.stroke();
        }
        context.restore();
    },
};

const cursorOverlayPlugin = {
    id: 'cursorOverlay',
    afterDatasetsDraw(chart: any) {
        const trend = chart._trendChartInstance as TrendChart | undefined;
        trend?.drawCursorOverlay(chart);
    },
};

export class TrendChart {
    private chart: Chart | null = null;
    private readonly canvas: HTMLCanvasElement | null;
    private boolCanvas: HTMLCanvasElement | null = null;
    private boolContainer: HTMLElement | null = null;
    private timeCanvas: HTMLCanvasElement | null = null;
    private timeContainer: HTMLElement | null = null;
    private tags: ChartTag[] = [];
    private axes: ChartAxis[] = [];
    private history = new Map<string, Point[]>();
    private axisRanges = new Map<string, { min: number; max: number }>();
    private timeWindowSeconds = 60;
    private viewOffsetSeconds = 0;
    private dragging = false;
    private lastPointerX = 0;
    private paused = false;
    private differential = false;
    private renderPending = false;
    private resizeObserver: ResizeObserver | null = null;
    private onDragStart?: () => void;
    private cursorsEnabled = false;
    private cursorA: number | null = null;
    private cursorB: number | null = null;
    private activeCursorDrag: 'A' | 'B' | null = null;
    private onCursorChange?: (meas: CursorMeasurement | null) => void;
    private handleWindowResize = () => {
        this.render();
    };

    constructor(
        canvasOrId: string | HTMLCanvasElement,
        boolBandOrId: string | HTMLElement,
        timeAxisOrId: string | HTMLElement,
        onDragStart?: () => void,
    ) {
        this.canvas =
            typeof canvasOrId === 'string'
                ? (document.getElementById(
                      canvasOrId,
                  ) as HTMLCanvasElement | null)
                : canvasOrId;

        const rawBool =
            typeof boolBandOrId === 'string'
                ? document.getElementById(boolBandOrId)
                : boolBandOrId;
        if (rawBool instanceof HTMLCanvasElement) {
            this.boolCanvas = rawBool;
            this.boolContainer = rawBool;
        } else if (rawBool) {
            this.boolContainer = rawBool;
            let c = rawBool.querySelector('canvas') as HTMLCanvasElement | null;
            if (!c) {
                c = document.createElement('canvas');
                rawBool.appendChild(c);
            }
            this.boolCanvas = c;
        }

        const rawTime =
            typeof timeAxisOrId === 'string'
                ? document.getElementById(timeAxisOrId)
                : timeAxisOrId;
        if (rawTime instanceof HTMLCanvasElement) {
            this.timeCanvas = rawTime;
            this.timeContainer = rawTime;
        } else if (rawTime) {
            this.timeContainer = rawTime;
            let c = rawTime.querySelector('canvas') as HTMLCanvasElement | null;
            if (!c) {
                c = document.createElement('canvas');
                rawTime.appendChild(c);
            }
            this.timeCanvas = c;
        }

        this.onDragStart = onDragStart;
        if (!this.canvas) return;

        this.chart = new Chart(this.canvas, {
            type: 'line',
            data: { datasets: [] },
            plugins: [secondGridPlugin, cursorOverlayPlugin],
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
                normalized: true,
                parsing: false,
                devicePixelRatio: 1,
                onResize: () => {
                    this.render();
                },
                layout: {
                    // Stable padding — X labels live outside Chart.js under the boolean band
                    padding: { top: 4, right: 4, bottom: 0, left: 4 },
                },
                interaction: { mode: 'nearest', axis: 'x', intersect: false },
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        mode: 'nearest',
                        axis: 'x',
                        intersect: false,
                        filter: (item: any, index: number, items: any[]) => {
                            return (
                                items.findIndex(
                                    (i) => i.datasetIndex === item.datasetIndex,
                                ) === index
                            );
                        },
                        callbacks: {
                            title: () => '',
                            label: (context: any) => {
                                let val = context.parsed.y;
                                if (typeof val === 'number') {
                                    val = Number(val.toFixed(3));
                                }
                                return `${context.dataset.label}: ${val}`;
                            },
                        },
                    },
                },
                scales: {
                    x: {
                        type: 'linear',
                        display: true,
                        grid: { display: false },
                        border: { display: false },
                        ticks: {
                            display: false, // drawn under boolean band
                            padding: 0,
                        },
                        afterFit: (axis: any) => {
                            // Collapse X-axis height so plot bottom is stable (no label-width jumps)
                            axis.height = 0;
                        },
                    },
                },
            } as any,
        });
        (this.chart as any)._trendChartInstance = this;

        if (typeof ResizeObserver !== 'undefined' && this.canvas) {
            this.resizeObserver = new ResizeObserver(() => {
                this.render();
            });
            this.resizeObserver.observe(this.canvas);
            if (this.canvas.parentElement) {
                this.resizeObserver.observe(this.canvas.parentElement);
            }
        }
        if (typeof window !== 'undefined') {
            window.addEventListener('resize', this.handleWindowResize);
        }

        this.canvas.style.cursor = 'crosshair';
        this.canvas.addEventListener('pointerdown', (event) => {
            if (event.button !== 0 || !this.canvas) return;

            const rect = this.canvas.getBoundingClientRect();
            const mouseX = event.clientX - rect.left;

            if (this.cursorsEnabled && this.chart) {
                const scale = this.chart.scales.x;
                if (scale && this.cursorA !== null && this.cursorB !== null) {
                    const xA = scale.getPixelForValue(this.cursorA);
                    const xB = scale.getPixelForValue(this.cursorB);
                    const distA = Math.abs(mouseX - xA);
                    const distB = Math.abs(mouseX - xB);

                    if (distA <= 12 && distA <= distB) {
                        this.activeCursorDrag = 'A';
                        try {
                            this.canvas.setPointerCapture(event.pointerId);
                        } catch {}
                        event.preventDefault();
                        return;
                    } else if (distB <= 12) {
                        this.activeCursorDrag = 'B';
                        try {
                            this.canvas.setPointerCapture(event.pointerId);
                        } catch {}
                        event.preventDefault();
                        return;
                    }
                }
            }

            this.activeCursorDrag = null;
            this.dragging = true;
            this.lastPointerX = event.clientX;
            try {
                this.canvas.setPointerCapture(event.pointerId);
            } catch {
                /* ignore */
            }
            this.canvas.style.cursor = 'grabbing';
            this.onDragStart?.();
            event.preventDefault();
        });

        this.canvas.addEventListener('pointermove', (event) => {
            if (!this.canvas) return;
            const rect = this.canvas.getBoundingClientRect();
            const mouseX = event.clientX - rect.left;

            if (this.activeCursorDrag && this.chart) {
                const scale = this.chart.scales.x;
                const area = this.chart.chartArea;
                if (scale && area) {
                    const clampedX = Math.max(area.left, Math.min(area.right, mouseX));
                    const val = scale.getValueForPixel(clampedX);
                    const newTime = Math.round(val ?? clampedX);
                    if (this.activeCursorDrag === 'A') {
                        this.cursorA = newTime;
                    } else if (this.activeCursorDrag === 'B') {
                        this.cursorB = newTime;
                    }
                    this.render();
                    this.emitMeasurementUpdate();
                }
                return;
            }

            if (this.dragging) {
                const delta = event.clientX - this.lastPointerX;
                this.lastPointerX = event.clientX;
                this.viewOffsetSeconds -=
                    (delta / Math.max(1, this.canvas.clientWidth)) *
                    this.timeWindowSeconds;
                this.render();
                return;
            }

            // Hover cursor styling for measurement cursors
            if (this.cursorsEnabled && this.chart) {
                const scale = this.chart.scales.x;
                if (scale && this.cursorA !== null && this.cursorB !== null) {
                    const xA = scale.getPixelForValue(this.cursorA);
                    const xB = scale.getPixelForValue(this.cursorB);
                    if (Math.abs(mouseX - xA) <= 8 || Math.abs(mouseX - xB) <= 8) {
                        this.canvas.style.cursor = 'ew-resize';
                        return;
                    }
                }
            }
            this.canvas.style.cursor = 'crosshair';
        });

        const release = (event: PointerEvent) => {
            this.activeCursorDrag = null;
            this.dragging = false;
            try {
                if (this.canvas?.hasPointerCapture(event.pointerId)) {
                    this.canvas.releasePointerCapture(event.pointerId);
                }
            } catch {
                /* ignore */
            }
            if (this.canvas) this.canvas.style.cursor = 'crosshair';
        };
        this.canvas.addEventListener('pointerup', release);
        this.canvas.addEventListener('pointercancel', release);
        this.canvas.addEventListener('lostpointercapture', () => {
            this.activeCursorDrag = null;
            this.dragging = false;
            if (this.canvas) this.canvas.style.cursor = 'crosshair';
        });
    }

    public setTags(tags: ChartTag[], axes: ChartAxis[]) {
        const oldAxes = JSON.stringify(this.axes);
        const newAxes = JSON.stringify(axes);
        if (oldAxes !== newAxes) this.axisRanges.clear();
        this.tags = tags
            .filter((tag) => tag.enabled !== false)
            .map((tag) => ({ ...tag }));
        this.axes = axes.map((axis) => ({ ...axis }));
        const ids = new Set(this.tags.map((tag) => tag.id));
        for (const id of this.history.keys())
            if (!ids.has(id)) this.history.delete(id);
        this.render();
    }

    public setWindow(seconds: number) {
        this.timeWindowSeconds = Math.min(86400, Math.max(30, seconds));
        this.render();
    }

    public setPaused(paused: boolean) {
        this.paused = paused;
        if (!paused) {
            this.viewOffsetSeconds = 0;
            this.render();
        }
    }

    private interpolationMode = 'Line';

    public setInterpolation(mode: string) {
        this.interpolationMode = mode;
        this.differential = mode === 'Differential';
        this.render();
    }

    public addDataPoint(id: string, timestamp: string, value: number) {
        if (!Number.isFinite(value)) return;
        const time = Date.parse(timestamp) || Date.now();
        const points = this.history.get(id) ?? [];
        points.push({ timestamp: time, value });

        // When in live mode (viewOffset == 0), retain only current display window plus margin
        if (this.viewOffsetSeconds === 0) {
            const cutoff = time - this.timeWindowSeconds * 1500;
            while (points.length > 0 && points[0].timestamp < cutoff) {
                points.shift();
            }
        } else if (points.length > 50000) {
            points.splice(0, 5000);
        }

        this.history.set(id, points);
        if (!this.paused) this.render();
    }

    private fetchHistoryPending = false;
    private fetchTimer: number | null = null;

    private scheduleHistoryFetch(startMs: number, endMs: number) {
        if (typeof window === 'undefined') return;
        const api = (window as any).go?.backend?.App;
        if (!api?.GetHistoryRange) return;

        if (this.fetchTimer !== null) {
            window.clearTimeout(this.fetchTimer);
        }

        this.fetchTimer = window.setTimeout(async () => {
            this.fetchTimer = null;
            if (this.fetchHistoryPending) return;
            this.fetchHistoryPending = true;

            try {
                const tagIds = this.tags.map((t) => t.id);
                if (tagIds.length === 0) return;

                // Add 10% padding around start and end
                const span = Math.max(1000, endMs - startMs);
                const pad = span * 0.1;
                const queryStart = Math.max(0, Math.round(startMs - pad));
                const queryEnd = Math.round(endMs + pad);

                const res: Record<string, Array<{ t: number; v: number }>> =
                    await api.GetHistoryRange(tagIds, queryStart, queryEnd);

                if (res) {
                    for (const [tagId, pts] of Object.entries(res)) {
                        if (Array.isArray(pts)) {
                            this.history.set(
                                tagId,
                                pts.map((p) => ({
                                    timestamp: p.t,
                                    value: p.v,
                                })),
                            );
                        }
                    }
                    this.render();
                }
            } catch {
                /* ignore fetch error */
            } finally {
                this.fetchHistoryPending = false;
            }
        }, 40);
    }

    public setCursorsEnabled(enabled: boolean) {
        this.cursorsEnabled = enabled;
        if (enabled && (this.cursorA === null || this.cursorB === null)) {
            this.fitCursorsToWindow();
        } else {
            this.render();
            this.emitMeasurementUpdate();
        }
    }

    public fitCursorsToWindow() {
        const latest = this.latestTimestamp();
        const end = latest + this.viewOffsetSeconds * 1000;
        const start = end - this.timeWindowSeconds * 1000;
        this.cursorA = Math.round(start + (end - start) * 0.25);
        this.cursorB = Math.round(start + (end - start) * 0.75);
        this.render();
        this.emitMeasurementUpdate();
    }

    public setOnCursorChange(cb?: (meas: CursorMeasurement | null) => void) {
        this.onCursorChange = cb;
        this.emitMeasurementUpdate();
    }

    public getMeasurements(): CursorMeasurement | null {
        if (
            !this.cursorsEnabled ||
            this.cursorA === null ||
            this.cursorB === null
        ) {
            return null;
        }
        const deltaTMs = this.cursorB - this.cursorA;
        const deltaTSec = deltaTMs / 1000;
        const tagStats: Record<string, TagStats> = {};

        for (const tag of this.tags) {
            const points = this.history.get(tag.id) ?? [];
            tagStats[tag.id] = calculateTagStats(
                points,
                this.cursorA,
                this.cursorB,
            );
        }

        return {
            cursorA: this.cursorA,
            cursorB: this.cursorB,
            deltaTSec,
            deltaTMs,
            tags: tagStats,
        };
    }

    private emitMeasurementUpdate() {
        if (this.onCursorChange) {
            this.onCursorChange(this.getMeasurements());
        }
    }

    public drawCursorOverlay(chart: any) {
        if (!this.cursorsEnabled) return;
        const scale = chart.scales.x;
        const area = chart.chartArea;
        if (!scale || !area) return;

        const min = Number(scale.min);
        const max = Number(scale.max);
        if (this.cursorA === null) this.cursorA = Math.round(min + (max - min) * 0.25);
        if (this.cursorB === null) this.cursorB = Math.round(min + (max - min) * 0.75);

        const xA = scale.getPixelForValue(this.cursorA);
        const xB = scale.getPixelForValue(this.cursorB);
        const ctx = chart.ctx as CanvasRenderingContext2D;
        ctx.save();

        // Highlight band between A and B
        const leftX = Math.max(area.left, Math.min(xA, xB));
        const rightX = Math.min(area.right, Math.max(xA, xB));
        if (rightX > leftX) {
            ctx.fillStyle = 'rgba(56, 189, 248, 0.07)';
            ctx.fillRect(leftX, area.top, rightX - leftX, area.bottom - area.top);
        }

        // Draw Cursor A
        if (xA >= area.left && xA <= area.right) {
            ctx.beginPath();
            ctx.setLineDash([4, 2]);
            ctx.strokeStyle = '#38BDF8';
            ctx.lineWidth = 1.5;
            ctx.moveTo(xA, area.top);
            ctx.lineTo(xA, area.bottom);
            ctx.stroke();

            // Top handle badge A
            ctx.setLineDash([]);
            ctx.fillStyle = '#0284C7';
            ctx.fillRect(xA - 10, area.top + 2, 20, 14);
            ctx.fillStyle = '#FFFFFF';
            ctx.font = 'bold 10px sans-serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('A', xA, area.top + 9);

            // Intersection markers for active analog tags
            for (const tag of this.tags) {
                if (tag.dataType === 'Bool') continue;
                const valA = interpolateValue(
                    this.history.get(tag.id) ?? [],
                    this.cursorA,
                );
                if (valA !== null) {
                    const axes = this.getUsedAxes(this.tags.filter(t => t.dataType !== 'Bool'));
                    const axisIdx = Math.max(0, axes.findIndex(a => a.name === tag.yAxis));
                    const yScale = chart.scales[`yAxis${axisIdx}`];
                    if (yScale) {
                        const yPix = yScale.getPixelForValue(valA);
                        if (yPix >= area.top && yPix <= area.bottom) {
                            ctx.beginPath();
                            ctx.arc(xA, yPix, 4, 0, 2 * Math.PI);
                            ctx.fillStyle = tag.color || '#38BDF8';
                            ctx.fill();
                            ctx.strokeStyle = '#FFFFFF';
                            ctx.lineWidth = 1.5;
                            ctx.stroke();
                        }
                    }
                }
            }
        }

        // Draw Cursor B
        if (xB >= area.left && xB <= area.right) {
            ctx.beginPath();
            ctx.setLineDash([4, 2]);
            ctx.strokeStyle = '#FBBF24';
            ctx.lineWidth = 1.5;
            ctx.moveTo(xB, area.top);
            ctx.lineTo(xB, area.bottom);
            ctx.stroke();

            // Top handle badge B
            ctx.setLineDash([]);
            ctx.fillStyle = '#D97706';
            ctx.fillRect(xB - 10, area.top + 2, 20, 14);
            ctx.fillStyle = '#FFFFFF';
            ctx.font = 'bold 10px sans-serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('B', xB, area.top + 9);

            // Intersection markers for active analog tags
            for (const tag of this.tags) {
                if (tag.dataType === 'Bool') continue;
                const valB = interpolateValue(
                    this.history.get(tag.id) ?? [],
                    this.cursorB,
                );
                if (valB !== null) {
                    const axes = this.getUsedAxes(this.tags.filter(t => t.dataType !== 'Bool'));
                    const axisIdx = Math.max(0, axes.findIndex(a => a.name === tag.yAxis));
                    const yScale = chart.scales[`yAxis${axisIdx}`];
                    if (yScale) {
                        const yPix = yScale.getPixelForValue(valB);
                        if (yPix >= area.top && yPix <= area.bottom) {
                            ctx.beginPath();
                            ctx.arc(xB, yPix, 4, 0, 2 * Math.PI);
                            ctx.fillStyle = tag.color || '#FBBF24';
                            ctx.fill();
                            ctx.strokeStyle = '#FFFFFF';
                            ctx.lineWidth = 1.5;
                            ctx.stroke();
                        }
                    }
                }
            }
        }

        ctx.restore();
    }

    public clear() {
        this.history.clear();
        this.axisRanges.clear();
        this.viewOffsetSeconds = 0;
        this.fixedIdleTime = Date.now();
        if (this.fetchTimer !== null && typeof window !== 'undefined') {
            window.clearTimeout(this.fetchTimer);
            this.fetchTimer = null;
        }
        const api =
            typeof window !== 'undefined'
                ? (window as any).go?.backend?.App
                : null;
        api?.ClearHistory?.();
        this.render();
        this.emitMeasurementUpdate();
    }

    public destroy() {
        if (this.fetchTimer !== null && typeof window !== 'undefined') {
            window.clearTimeout(this.fetchTimer);
            this.fetchTimer = null;
        }
        this.resizeObserver?.disconnect();
        this.resizeObserver = null;
        if (typeof window !== 'undefined') {
            window.removeEventListener('resize', this.handleWindowResize);
        }
        this.chart?.destroy();
        this.chart = null;
    }

    private render() {
        if (this.renderPending) return;
        this.renderPending = true;
        requestAnimationFrame(() => {
            this.renderPending = false;
            this.doRender();
        });
    }

    private doRender() {
        if (!this.chart) return;
        const latest = this.latestTimestamp();
        const end = latest + this.viewOffsetSeconds * 1000;
        const start = end - this.timeWindowSeconds * 1000;

        // If panning into history, fetch requested range from backend
        if (this.viewOffsetSeconds < 0) {
            this.scheduleHistoryFetch(start, end);
        }

        const analogTags = this.tags.filter((tag) => tag.dataType !== 'Bool');
        const axes = this.getUsedAxes(analogTags);
        const datasets = analogTags.map((tag) => {
            const axisIndex = Math.max(
                0,
                axes.findIndex((axis) => axis.name === tag.yAxis),
            );
            const points = (this.history.get(tag.id) ?? []).filter(
                (point) => point.timestamp >= start && point.timestamp <= end,
            );
            let tension = 0;
            let stepped: boolean | 'before' | 'after' | 'middle' = false;
            if (this.interpolationMode === 'Cubic') tension = 0.4;
            if (this.interpolationMode === 'Step') stepped = 'before';

            return {
                label: tag.name,
                data: points.map((point, index) => ({
                    x: point.timestamp,
                    y: this.differential
                        ? point.value -
                          (points[index - 1]?.value ?? point.value)
                        : point.value,
                })),
                borderColor: tag.color || '#93C5FD',
                backgroundColor: tag.color || '#93C5FD',
                borderWidth: 1,
                pointRadius: 0,
                pointHoverRadius: 2,
                pointHoverBorderWidth: 1,
                tension,
                stepped,
                fill: false,
                yAxisID: `yAxis${axisIndex}`,
            };
        });

        const scales: Record<string, any> = {
            x: {
                type: 'linear',
                min: start,
                max: end,
                display: true,
                grid: { display: false },
                border: { display: false },
                ticks: {
                    display: false,
                    padding: 0,
                },
                afterFit: (axis: any) => {
                    axis.height = 0;
                },
            },
        };
        axes.forEach((axis, index) => {
            const range = this.getAxisRange(
                axis,
                analogTags.filter((tag) => tag.yAxis === axis.name),
            );
            scales[`yAxis${index}`] = {
                type: 'linear',
                position: index % 2 === 0 ? 'left' : 'right',
                min: range.min,
                max: range.max,
                grid: {
                    drawOnChartArea: index === 0,
                    color: (ctx: any) =>
                        ctx.index % 2 !== 0
                            ? 'rgba(49, 65, 78, 0.5)'
                            : '#31414E',
                    borderDash: (ctx: any) =>
                        ctx.index % 2 !== 0 ? [4, 4] : [],
                },
                ticks: {
                    color: '#CBD5E1',
                    precision: 0,
                    maxTicksLimit: 21,
                    callback: function (value: number, index: number) {
                        return index % 2 !== 0 ? '' : value;
                    },
                },
            };
        });

        this.chart.data.datasets = datasets as any;
        (this.chart.options as any).scales = scales;
        this.chart.update('none');

        const area = this.chart.chartArea;
        this.renderBooleanBand(start, end, area);
        this.renderTimeAxis(start, end, area);
        this.emitMeasurementUpdate();
    }

    private fixedIdleTime = Date.now();

    private latestTimestamp(): number {
        let maxTs = 0;
        for (const points of this.history.values()) {
            if (points.length > 0) {
                const ts = points[points.length - 1].timestamp;
                if (ts > maxTs) maxTs = ts;
            }
        }
        if (maxTs > 0) {
            return maxTs;
        }
        return this.fixedIdleTime;
    }

    private getUsedAxes(tags: ChartTag[]): ChartAxis[] {
        const names = [
            ...new Set(tags.map((tag) => tag.yAxis).filter(Boolean)),
        ];
        const configured = names.map(
            (name) =>
                this.axes.find((axis) => axis.name === name) ?? {
                    name,
                    minimum: 0,
                    maximum: 100,
                    autoScale: true,
                },
        );
        return configured.length > 0
            ? configured
            : [{ name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true }];
    }

    private getAxisRange(axis: ChartAxis, tags: ChartTag[]) {
        let range = this.axisRanges.get(axis.name) ?? {
            min: axis.minimum,
            max: axis.maximum,
        };
        if (range.max <= range.min) range = { min: 0, max: 100 };
        const values = tags.flatMap((tag) =>
            (this.history.get(tag.id) ?? []).map((point) => point.value),
        );
        if (values.length > 0) {
            const pad = Math.max(
                0.001,
                (Math.max(...values) - Math.min(...values)) * 0.08,
            );
            if (Math.min(...values) < range.min)
                range.min = Math.min(...values) - pad;
            if (Math.max(...values) > range.max)
                range.max = Math.max(...values) + pad;
        }
        this.axisRanges.set(axis.name, range);
        return range;
    }

    /** Plot-area geometry shared by boolean band + time labels. */
    private plotLayout(
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.canvas || !area) {
            return { left: 0, right: 0, width: 0, canvasW: 0 };
        }
        const canvasW = this.canvas.clientWidth || this.canvas.width;
        return {
            left: area.left,
            right: canvasW - area.right,
            width: area.width,
            canvasW,
        };
    }

    private getCanvasContext(
        canvas: HTMLCanvasElement,
        width: number,
        height: number,
    ): CanvasRenderingContext2D | null {
        const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
        const targetW = Math.max(1, Math.round(width * dpr));
        const targetH = Math.max(1, Math.round(height * dpr));

        if (canvas.width !== targetW || canvas.height !== targetH) {
            canvas.width = targetW;
            canvas.height = targetH;
        }

        const ctx = canvas.getContext('2d');
        if (!ctx) return null;

        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        return ctx;
    }

    private renderBooleanBand(
        start: number,
        end: number,
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.boolCanvas) return;
        const tags = this.tags
            .filter((tag) => tag.dataType === 'Bool')
            .slice(0, 8);

        const isHidden = tags.length === 0;
        this.boolCanvas.classList.toggle('hidden', isHidden);
        if (this.boolContainer && this.boolContainer !== this.boolCanvas) {
            this.boolContainer.classList.toggle('hidden', isHidden);
        }
        if (isHidden) return;

        const rowHeight = 24;
        const totalHeight = tags.length * rowHeight;
        const layout = this.plotLayout(area);
        const canvasW = layout.canvasW || this.canvas?.clientWidth || 800;

        this.boolCanvas.style.height = `${totalHeight}px`;
        const ctx = this.getCanvasContext(this.boolCanvas, canvasW, totalHeight);
        if (!ctx) return;

        ctx.clearRect(0, 0, canvasW, totalHeight);

        // Background
        ctx.fillStyle = '#101820';
        ctx.fillRect(0, 0, canvasW, totalHeight);

        const span = Math.max(1, end - start);
        const spanSeconds = span / 1000;
        const { stepMs, majorMs } = chooseXGridIntervals(spanSeconds);
        const firstGrid = Math.ceil(start / stepMs) * stepMs;

        tags.forEach((tag, index) => {
            const topY = index * rowHeight;

            // Bottom border for each row
            ctx.strokeStyle = '#536572';
            ctx.lineWidth = 1;
            ctx.setLineDash([]);
            ctx.beginPath();
            ctx.moveTo(0, topY + rowHeight - 0.5);
            ctx.lineTo(canvasW, topY + rowHeight - 0.5);
            ctx.stroke();

            // Tag Name in left gutter
            if (layout.left > 0) {
                ctx.save();
                ctx.beginPath();
                ctx.rect(0, topY, layout.left, rowHeight);
                ctx.clip();
                ctx.font =
                    '11px "Segoe UI", -apple-system, BlinkMacSystemFont, Roboto, sans-serif';
                ctx.fillStyle = '#CBD5E1';
                ctx.textAlign = 'left';
                ctx.textBaseline = 'middle';
                ctx.fillText(tag.name, 6, topY + rowHeight / 2);
                ctx.restore();
            }

            // Grid lines and data blocks in track area
            if (layout.width > 0) {
                ctx.save();
                ctx.beginPath();
                ctx.rect(layout.left, topY, layout.width, rowHeight);
                ctx.clip();

                // Vertical grid lines
                for (
                    let timestamp = firstGrid;
                    timestamp <= end;
                    timestamp += stepMs
                ) {
                    const x =
                        layout.left + ((timestamp - start) / span) * layout.width;
                    if (x < layout.left || x > layout.left + layout.width) continue;

                    const major = isMajorTick(timestamp, majorMs, stepMs);
                    ctx.strokeStyle = major ? '#5B7182' : '#31414E';
                    ctx.lineWidth = major ? 1 : 0.8;
                    ctx.setLineDash(major ? [] : [4, 4]);

                    ctx.beginPath();
                    ctx.moveTo(Math.round(x) + 0.5, topY);
                    ctx.lineTo(Math.round(x) + 0.5, topY + rowHeight);
                    ctx.stroke();
                }

                // Data high state blocks
                const points = this.history.get(tag.id) ?? [];
                ctx.fillStyle = tag.color || '#93C5FD';
                ctx.setLineDash([]);

                const barTop = topY + 6;
                const barHeight = 12;

                points.forEach((point, pIndex) => {
                    const next = points[pIndex + 1]?.timestamp ?? end;
                    if (
                        point.value <= 0.5 ||
                        next <= start ||
                        point.timestamp >= end
                    ) {
                        return;
                    }

                    const clampedStart = Math.max(point.timestamp, start);
                    const clampedEnd = Math.min(next, end);

                    const x1 =
                        layout.left +
                        ((clampedStart - start) / span) * layout.width;
                    const x2 =
                        layout.left +
                        ((clampedEnd - start) / span) * layout.width;
                    const blockWidth = Math.max(1, x2 - x1);

                    ctx.fillRect(x1, barTop, blockWidth, barHeight);
                });

                ctx.restore();
            }
        });

        // Draw Cursor guidelines across boolean band
        if (this.cursorsEnabled && layout.width > 0) {
            ctx.save();
            ctx.beginPath();
            ctx.rect(layout.left, 0, layout.width, totalHeight);
            ctx.clip();

            if (this.cursorA !== null && this.cursorA >= start && this.cursorA <= end) {
                const xA = layout.left + ((this.cursorA - start) / span) * layout.width;
                ctx.beginPath();
                ctx.setLineDash([4, 2]);
                ctx.strokeStyle = '#38BDF8';
                ctx.lineWidth = 1.5;
                ctx.moveTo(xA, 0);
                ctx.lineTo(xA, totalHeight);
                ctx.stroke();
            }

            if (this.cursorB !== null && this.cursorB >= start && this.cursorB <= end) {
                const xB = layout.left + ((this.cursorB - start) / span) * layout.width;
                ctx.beginPath();
                ctx.setLineDash([4, 2]);
                ctx.strokeStyle = '#FBBF24';
                ctx.lineWidth = 1.5;
                ctx.moveTo(xB, 0);
                ctx.lineTo(xB, totalHeight);
                ctx.stroke();
            }
            ctx.restore();
        }
    }

    private renderTimeAxis(
        start: number,
        end: number,
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.timeCanvas) return;
        const layout = this.plotLayout(area);
        const canvasW = layout.canvasW || this.canvas?.clientWidth || 800;
        const height = 22;

        this.timeCanvas.style.height = `${height}px`;
        const ctx = this.getCanvasContext(this.timeCanvas, canvasW, height);
        if (!ctx) return;

        ctx.clearRect(0, 0, canvasW, height);

        // Background
        ctx.fillStyle = '#0C1118';
        ctx.fillRect(0, 0, canvasW, height);

        // Top border line
        ctx.strokeStyle = '#31414E';
        ctx.lineWidth = 1;
        ctx.setLineDash([]);
        ctx.beginPath();
        ctx.moveTo(0, 0.5);
        ctx.lineTo(canvasW, 0.5);
        ctx.stroke();

        if (layout.width <= 0) return;

        const span = Math.max(1, end - start);
        const spanSeconds = span / 1000;
        const { majorMs } = chooseXGridIntervals(spanSeconds);

        ctx.save();
        ctx.beginPath();
        ctx.rect(layout.left, 0, layout.width, height);
        ctx.clip();

        ctx.font =
            '10px ui-monospace, "Cascadia Mono", "Segoe UI Mono", Consolas, monospace';
        ctx.fillStyle = '#CBD5E1';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';

        const first = Math.ceil(start / majorMs) * majorMs;
        for (
            let timestamp = first;
            timestamp <= end + 0.5;
            timestamp += majorMs
        ) {
            const x =
                layout.left + ((timestamp - start) / span) * layout.width;
            if (x < layout.left - 20 || x > layout.left + layout.width + 20)
                continue;
            const timeStr = formatTickTime(timestamp);
            ctx.fillText(timeStr, x, height / 2);
        }

        // Draw Cursor indicators on time axis
        if (this.cursorsEnabled) {
            if (this.cursorA !== null && this.cursorA >= start && this.cursorA <= end) {
                const xA = layout.left + ((this.cursorA - start) / span) * layout.width;
                ctx.fillStyle = '#0284C7';
                ctx.fillRect(xA - 8, 3, 16, 16);
                ctx.fillStyle = '#FFFFFF';
                ctx.font = 'bold 9px sans-serif';
                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                ctx.fillText('A', xA, 11);
            }

            if (this.cursorB !== null && this.cursorB >= start && this.cursorB <= end) {
                const xB = layout.left + ((this.cursorB - start) / span) * layout.width;
                ctx.fillStyle = '#D97706';
                ctx.fillRect(xB - 8, 3, 16, 16);
                ctx.fillStyle = '#FFFFFF';
                ctx.font = 'bold 9px sans-serif';
                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                ctx.fillText('B', xB, 11);
            }
        }

        ctx.restore();
    }
}
