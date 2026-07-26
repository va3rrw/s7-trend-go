import Chart from 'chart.js/auto';

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

interface Point {
    timestamp: number;
    value: number;
}

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

export class TrendChart {
    private chart: Chart | null = null;
    private readonly canvas: HTMLCanvasElement | null;
    private readonly booleanBand: HTMLElement | null;
    private readonly timeAxis: HTMLElement | null;
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
    private onDragStart?: () => void;

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
        this.booleanBand =
            typeof boolBandOrId === 'string'
                ? document.getElementById(boolBandOrId)
                : boolBandOrId;
        this.timeAxis =
            typeof timeAxisOrId === 'string'
                ? document.getElementById(timeAxisOrId)
                : timeAxisOrId;
        this.onDragStart = onDragStart;
        if (!this.canvas) return;

        this.chart = new Chart(this.canvas, {
            type: 'line',
            data: { datasets: [] },
            plugins: [secondGridPlugin],
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
                normalized: true,
                parsing: false,
                devicePixelRatio: 1,
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

        this.canvas.style.cursor = 'crosshair';
        this.canvas.addEventListener('pointerdown', (event) => {
            if (event.button !== 0) return;
            this.dragging = true;
            this.lastPointerX = event.clientX;
            try {
                this.canvas?.setPointerCapture(event.pointerId);
            } catch {
                /* ignore */
            }
            if (this.canvas) this.canvas.style.cursor = 'grabbing';
            this.onDragStart?.();
            event.preventDefault();
        });
        this.canvas.addEventListener('pointermove', (event) => {
            if (!this.dragging || !this.canvas) return;
            const delta = event.clientX - this.lastPointerX;
            this.lastPointerX = event.clientX;
            this.viewOffsetSeconds -=
                (delta / Math.max(1, this.canvas.clientWidth)) *
                this.timeWindowSeconds;
            this.render();
        });
        const release = (event: PointerEvent) => {
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
        if (points.length > 100000) {
            points.splice(0, 10000);
        }
        this.history.set(id, points);
        if (!this.paused) this.render();
    }

    public clear() {
        this.history.clear();
        this.axisRanges.clear();
        this.viewOffsetSeconds = 0;
        this.render();
    }

    public destroy() {
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
    }

    private latestTimestamp(): number {
        let latest = Date.now();
        for (const points of this.history.values()) {
            if (points.length > 0)
                latest = Math.max(latest, points[points.length - 1].timestamp);
        }
        return latest;
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

    private renderBooleanBand(
        start: number,
        end: number,
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.booleanBand) return;
        const tags = this.tags
            .filter((tag) => tag.dataType === 'Bool')
            .slice(0, 8);
        this.booleanBand.innerHTML = '';
        this.booleanBand.classList.toggle('hidden', tags.length === 0);
        if (tags.length === 0) return;

        const layout = this.plotLayout(area);
        const span = Math.max(1, end - start);
        const spanSeconds = span / 1000;
        const { stepMs, majorMs } = chooseXGridIntervals(spanSeconds);

        const grid = document.createElement('div');
        grid.className = 'boolean-grid';

        tags.forEach((tag) => {
            const row = document.createElement('div');
            row.className = 'boolean-row';

            // Name sits in the left Y-axis gutter so the track aligns with the plot
            const name = document.createElement('span');
            name.className = 'boolean-name';
            name.style.width = `${layout.left}px`;
            name.style.flex = `0 0 ${layout.left}px`;
            name.textContent = tag.name;
            row.appendChild(name);

            const track = document.createElement('div');
            track.className = 'boolean-track';
            track.style.width = `${layout.width}px`;
            track.style.flex = `0 0 ${layout.width}px`;

            // Vertical grid lines — same absolute times as the main chart
            const first = Math.ceil(start / stepMs) * stepMs;
            for (let timestamp = first; timestamp <= end; timestamp += stepMs) {
                const pct = ((timestamp - start) / span) * 100;
                if (pct < 0 || pct > 100) continue;
                const line = document.createElement('span');
                const major = isMajorTick(timestamp, majorMs, stepMs);
                line.className = major
                    ? 'boolean-vgrid major'
                    : 'boolean-vgrid minor';
                line.style.left = `${pct}%`;
                track.appendChild(line);
            }

            const points = this.history.get(tag.id) ?? [];
            points.forEach((point, index) => {
                const next = points[index + 1]?.timestamp ?? end;
                if (
                    point.value <= 0.5 ||
                    next <= start ||
                    point.timestamp >= end
                )
                    return;
                const block = document.createElement('span');
                block.className = 'boolean-high';
                block.style.backgroundColor = tag.color || '#93C5FD';
                block.style.left = `${Math.max(0, ((point.timestamp - start) / span) * 100)}%`;
                block.style.width = `${Math.max(
                    0.15,
                    Math.min(
                        100,
                        ((next - Math.max(point.timestamp, start)) / span) *
                            100,
                    ),
                )}%`;
                track.appendChild(block);
            });
            row.appendChild(track);

            // Right gutter matches right Y-axis space
            if (layout.right > 0) {
                const rightPad = document.createElement('span');
                rightPad.className = 'boolean-right-pad';
                rightPad.style.flex = `0 0 ${layout.right}px`;
                rightPad.style.width = `${layout.right}px`;
                row.appendChild(rightPad);
            }

            grid.appendChild(row);
        });
        this.booleanBand.appendChild(grid);
    }

    private renderTimeAxis(
        start: number,
        end: number,
        area: { left: number; right: number; width: number } | undefined,
    ) {
        if (!this.timeAxis) return;
        this.timeAxis.innerHTML = '';

        const layout = this.plotLayout(area);
        if (layout.width <= 0) return;

        const span = Math.max(1, end - start);
        const spanSeconds = span / 1000;
        const { majorMs } = chooseXGridIntervals(spanSeconds);

        const gutter = document.createElement('div');
        gutter.className = 'time-axis-gutter';
        gutter.style.width = `${layout.left}px`;
        gutter.style.flex = `0 0 ${layout.left}px`;
        this.timeAxis.appendChild(gutter);

        const track = document.createElement('div');
        track.className = 'time-axis-track';
        track.style.width = `${layout.width}px`;
        track.style.flex = `0 0 ${layout.width}px`;

        const first = Math.ceil(start / majorMs) * majorMs;
        for (
            let timestamp = first;
            timestamp <= end + 0.5;
            timestamp += majorMs
        ) {
            const pct = ((timestamp - start) / span) * 100;
            if (pct < -2 || pct > 102) continue;
            const label = document.createElement('span');
            label.className = 'time-axis-label';
            label.textContent = formatTickTime(timestamp);
            label.style.left = `${pct}%`;
            track.appendChild(label);
        }
        this.timeAxis.appendChild(track);

        if (layout.right > 0) {
            const rightPad = document.createElement('div');
            rightPad.className = 'time-axis-gutter';
            rightPad.style.width = `${layout.right}px`;
            rightPad.style.flex = `0 0 ${layout.right}px`;
            this.timeAxis.appendChild(rightPad);
        }
    }
}
