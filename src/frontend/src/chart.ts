import Chart from 'chart.js/auto';
import {
    calculateTagStats,
    type Point,
    type CursorMeasurement,
    type TagStats,
} from './measurement';
import {
    CHART_THEME,
    CHART_LAYOUT,
    CHART_LIMITS,
} from './chartConstants';
import {
    chooseXGridIntervals,
    isMajorTick,
    mergePoints,
    getCanvasContext,
    computePlotLayout,
    renderCursorOverlay,
    drawBooleanBand,
    drawTimeAxis,
} from './canvasHelpers';

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
            context.strokeStyle = major
                ? CHART_THEME.gridMajor
                : CHART_THEME.gridMinor;
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
    private baseTimeWindowSeconds: number = CHART_LIMITS.defaultTimeWindowSec;
    private timeWindowSeconds: number = CHART_LIMITS.defaultTimeWindowSec;
    private viewOffsetSeconds = 0;
    private lastLiveTimestamp = 0;
    private pausedAnchorTime: number | null = null;
    private dragging = false;
    private lastPointerX = 0;
    private paused = false;
    private renderPending = false;
    private resizeObserver: ResizeObserver | null = null;
    private onDragStart?: () => void;
    private onWindowChange?: (seconds: number) => void;
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
                            display: false,
                            padding: 0,
                        },
                        afterFit: (axis: any) => {
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

                    if (distA <= CHART_LAYOUT.cursorHitTolerance && distA <= distB) {
                        this.activeCursorDrag = 'A';
                        try {
                            this.canvas.setPointerCapture(event.pointerId);
                        } catch {}
                        event.preventDefault();
                        return;
                    } else if (distB <= CHART_LAYOUT.cursorHitTolerance) {
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
                if (this.viewOffsetSeconds > 0) {
                    this.viewOffsetSeconds = 0;
                }
                this.render();
                return;
            }

            // Hover cursor styling for measurement cursors
            if (this.cursorsEnabled && this.chart) {
                const scale = this.chart.scales.x;
                if (scale && this.cursorA !== null && this.cursorB !== null) {
                    const xA = scale.getPixelForValue(this.cursorA);
                    const xB = scale.getPixelForValue(this.cursorB);
                    if (
                        Math.abs(mouseX - xA) <= CHART_LAYOUT.cursorHoverTolerance ||
                        Math.abs(mouseX - xB) <= CHART_LAYOUT.cursorHoverTolerance
                    ) {
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

        this.canvas.addEventListener(
            'wheel',
            (event: WheelEvent) => {
                if (!this.canvas || !this.chart) return;
                event.preventDefault();

                const rect = this.canvas.getBoundingClientRect();
                const mouseX = event.clientX - rect.left;
                const area = this.chart.chartArea || {
                    left: 0,
                    right: this.canvas.clientWidth,
                    width: this.canvas.clientWidth,
                };
                const plotLeft = area.left ?? 0;
                const plotWidth = Math.max(
                    1,
                    area.width ||
                        (area.right !== undefined ? area.right - plotLeft : 0) ||
                        this.canvas.clientWidth,
                );
                const mouseRatio = Math.max(
                    0,
                    Math.min(1, (mouseX - plotLeft) / plotWidth),
                );

                const oldWindow = this.timeWindowSeconds;
                const factor =
                    event.deltaY < 0
                        ? CHART_LIMITS.zoomInFactor
                        : CHART_LIMITS.zoomOutFactor;
                let newWindow = Math.round(oldWindow * factor);
                newWindow = Math.min(
                    CHART_LIMITS.maxTimeWindowSec,
                    Math.max(CHART_LIMITS.minTimeWindowSec, newWindow),
                );
                if (newWindow === oldWindow) return;

                const anchor =
                    this.paused && this.pausedAnchorTime !== null
                        ? this.pausedAnchorTime
                        : this.latestTimestamp();
                const oldEnd = anchor + this.viewOffsetSeconds * 1000;
                const oldStart = oldEnd - oldWindow * 1000;
                const tMouse = oldStart + mouseRatio * (oldEnd - oldStart);

                const newStart = tMouse - mouseRatio * (newWindow * 1000);
                const newEnd = newStart + newWindow * 1000;
                let newOffset = (newEnd - anchor) / 1000;

                if (newOffset > 0) {
                    newOffset = 0;
                }

                // Any wheel zoom while running live pauses the chart for review
                if (!this.paused) {
                    this.paused = true;
                    this.pausedAnchorTime = anchor;
                    this.onDragStart?.();
                }

                this.timeWindowSeconds = newWindow;
                this.viewOffsetSeconds = newOffset;
                this.onWindowChange?.(newWindow);
                this.render();
            },
            { passive: false },
        );
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
        const val = Math.min(
            CHART_LIMITS.maxTimeWindowSec,
            Math.max(CHART_LIMITS.minTimeWindowSec, seconds),
        );
        this.baseTimeWindowSeconds = val;
        if (this.timeWindowSeconds === val) return;
        this.timeWindowSeconds = val;
        this.render();
    }

    public setPaused(paused: boolean) {
        if (this.paused === paused) return;
        this.paused = paused;
        if (paused) {
            this.pausedAnchorTime = this.latestTimestamp();
            this.viewOffsetSeconds = 0;
        } else {
            this.pausedAnchorTime = null;
            this.viewOffsetSeconds = 0;
            this.timeWindowSeconds = this.baseTimeWindowSeconds;
            this.onWindowChange?.(this.baseTimeWindowSeconds);
            this.render();
        }
    }

    private interpolationMode = 'Line';

    public setInterpolation(mode: string) {
        this.interpolationMode = mode;
        this.render();
    }

    public addDataPoint(id: string, timestamp: string, value: number) {
        if (!Number.isFinite(value)) return;
        const time = Date.parse(timestamp) || Date.now();
        this.lastLiveTimestamp = Math.max(this.lastLiveTimestamp, time);

        let points = this.history.get(id);
        if (!points) {
            points = [];
            this.history.set(id, points);
        }
        points.push({ timestamp: time, value });

        // Retain generous rolling buffer (50,000 points per tag ~1.4h at 100ms)
        if (points.length > CHART_LIMITS.pruneThreshold) {
            points.splice(
                0,
                points.length - CHART_LIMITS.maxHistoryCapacity,
            );
        }

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
                const span = Math.max(CHART_LIMITS.minFetchSpanMs, endMs - startMs);
                const pad = span * CHART_LIMITS.historyFetchPadRatio;
                const queryStart = Math.max(0, Math.round(startMs - pad));
                const queryEnd = Math.round(endMs + pad);

                const res: Record<string, Array<{ t: number; v: number }>> =
                    await api.GetHistoryRange(tagIds, queryStart, queryEnd);

                if (res) {
                    for (const [tagId, pts] of Object.entries(res)) {
                        if (Array.isArray(pts) && pts.length > 0) {
                            const existing = this.history.get(tagId) ?? [];
                            const fetched: Point[] = pts.map((p) => ({
                                timestamp: p.t,
                                value: p.v,
                            }));
                            this.history.set(
                                tagId,
                                mergePoints(existing, fetched),
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
        }, CHART_LIMITS.historyFetchDebounceMs);
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
        const anchor =
            this.paused && this.pausedAnchorTime !== null
                ? this.pausedAnchorTime
                : this.latestTimestamp();
        const end = anchor + this.viewOffsetSeconds * 1000;
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

    public setOnWindowChange(cb?: (seconds: number) => void) {
        this.onWindowChange = cb;
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
        const analogTags = this.tags.filter((t) => t.dataType !== 'Bool');
        const axes = this.getUsedAxes(analogTags);
        renderCursorOverlay(chart, {
            cursorsEnabled: this.cursorsEnabled,
            cursorA: this.cursorA,
            cursorB: this.cursorB,
            tags: this.tags,
            axes,
            history: this.history,
        });
    }

    public clear() {
        this.history.clear();
        this.axisRanges.clear();
        this.viewOffsetSeconds = 0;
        this.pausedAnchorTime = null;
        this.lastLiveTimestamp = 0;
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
        const anchor =
            this.paused && this.pausedAnchorTime !== null
                ? this.pausedAnchorTime
                : this.latestTimestamp();
        const end = anchor + this.viewOffsetSeconds * 1000;
        const start = end - this.timeWindowSeconds * 1000;

        // If panning or zooming into history beyond in-memory buffer, fetch from backend
        let earliestInMemory = Infinity;
        for (const tag of this.tags) {
            const pts = this.history.get(tag.id);
            if (pts && pts.length > 0) {
                earliestInMemory = Math.min(earliestInMemory, pts[0].timestamp);
            }
        }
        if (this.viewOffsetSeconds < 0 || start < earliestInMemory) {
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
                data: points.map((point) => ({
                    x: point.timestamp,
                    y: point.value,
                })),
                borderColor: tag.color || CHART_THEME.defaultSeries,
                backgroundColor: tag.color || CHART_THEME.defaultSeries,
                borderWidth: 1,
                pointRadius: 0,
                pointHoverRadius: CHART_LAYOUT.analogHoverRadius,
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
                            ? CHART_THEME.gridAlt
                            : CHART_THEME.gridMinor,
                    borderDash: (ctx: any) =>
                        ctx.index % 2 !== 0 ? [4, 4] : [],
                },
                ticks: {
                    color: CHART_THEME.textMuted,
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
        if (this.lastLiveTimestamp > 0) {
            return this.lastLiveTimestamp;
        }
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
                CHART_LIMITS.yAxisMinPad,
                (Math.max(...values) - Math.min(...values)) *
                    CHART_LIMITS.yAxisPadRatio,
            );
            if (Math.min(...values) < range.min)
                range.min = Math.min(...values) - pad;
            if (Math.max(...values) > range.max)
                range.max = Math.max(...values) + pad;
        }
        this.axisRanges.set(axis.name, range);
        return range;
    }

    private renderBooleanBand(
        start: number,
        end: number,
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.boolCanvas) return;
        const tags = this.tags.filter((tag) => tag.dataType === 'Bool');

        const isHidden = tags.length === 0;
        this.boolCanvas.classList.toggle('hidden', isHidden);
        if (this.boolContainer && this.boolContainer !== this.boolCanvas) {
            this.boolContainer.classList.toggle('hidden', isHidden);
        }
        if (isHidden) return;

        const rowHeight = CHART_LAYOUT.boolRowHeight;
        const totalHeight = tags.length * rowHeight;
        const layout = computePlotLayout(this.canvas, area);
        const canvasW = layout.canvasW || this.canvas?.clientWidth || 800;

        this.boolCanvas.style.height = `${totalHeight}px`;
        const ctx = getCanvasContext(this.boolCanvas, canvasW, totalHeight);
        if (!ctx) return;

        drawBooleanBand(ctx, {
            start,
            end,
            layout,
            boolTags: tags,
            history: this.history,
            cursorsEnabled: this.cursorsEnabled,
            cursorA: this.cursorA,
            cursorB: this.cursorB,
        });
    }

    private renderTimeAxis(
        start: number,
        end: number,
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.timeCanvas) return;
        const layout = computePlotLayout(this.canvas, area);
        const canvasW = layout.canvasW || this.canvas?.clientWidth || 800;
        const height = CHART_LAYOUT.timeAxisHeight;

        this.timeCanvas.style.height = `${height}px`;
        const ctx = getCanvasContext(this.timeCanvas, canvasW, height);
        if (!ctx) return;

        drawTimeAxis(ctx, {
            start,
            end,
            layout,
            cursorsEnabled: this.cursorsEnabled,
            cursorA: this.cursorA,
            cursorB: this.cursorB,
        });
    }
}
