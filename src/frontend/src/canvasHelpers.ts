import type { Point } from './measurement';
import { interpolateValue } from './measurement';
import type { ChartTag, ChartAxis } from './chart';
import {
    CHART_THEME,
    CHART_LAYOUT,
    CHART_LIMITS,
    CHART_FONTS,
    NICE_STEPS_SEC,
} from './chartConstants';

/** Largest nice step that is still <= rawSeconds (prefer denser grids). */
export function niceStepAtMost(rawSeconds: number): number {
    let best: number = NICE_STEPS_SEC[0];
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
export function chooseXGridIntervals(spanSeconds: number): {
    stepMs: number;
    majorMs: number;
} {
    const span = Math.max(1, spanSeconds);
    let majorSec = niceStepAtMost(span / 12);
    if (span / majorSec < 6 && majorSec > 1) {
        const idx = NICE_STEPS_SEC.indexOf(majorSec as any);
        if (idx > 0) majorSec = NICE_STEPS_SEC[idx - 1];
    }
    while (span / majorSec > 15) {
        const idx = NICE_STEPS_SEC.indexOf(majorSec as any);
        if (idx < 0 || idx >= NICE_STEPS_SEC.length - 1) break;
        majorSec = NICE_STEPS_SEC[idx + 1];
    }
    const minorSec = majorSec / 10;
    return { stepMs: minorSec * 1000, majorMs: majorSec * 1000 };
}

/** Fixed-width HH:mm:ss so label width never resizes the plot. */
export function formatTickTime(ms: number): string {
    const d = new Date(ms);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export function isMajorTick(
    timestamp: number,
    majorMs: number,
    stepMs: number,
): boolean {
    const rem = ((timestamp % majorMs) + majorMs) % majorMs;
    return rem < stepMs * 0.5 || rem > majorMs - stepMs * 0.5;
}

export function mergePoints(
    existing: Point[],
    fetched: Point[],
    maxCapacity: number = CHART_LIMITS.maxHistoryCapacity,
): Point[] {
    if (existing.length === 0) return fetched.slice(-maxCapacity);
    if (fetched.length === 0) return existing;

    if (fetched[fetched.length - 1].timestamp < existing[0].timestamp) {
        const combined = fetched.concat(existing);
        return combined.length > maxCapacity
            ? combined.slice(combined.length - maxCapacity)
            : combined;
    }
    if (existing[existing.length - 1].timestamp < fetched[0].timestamp) {
        const combined = existing.concat(fetched);
        return combined.length > maxCapacity
            ? combined.slice(combined.length - maxCapacity)
            : combined;
    }

    const map = new Map<number, number>();
    for (const p of existing) map.set(p.timestamp, p.value);
    for (const p of fetched) map.set(p.timestamp, p.value);
    const result: Point[] = [];
    for (const [timestamp, value] of map) {
        result.push({ timestamp, value });
    }
    result.sort((a, b) => a.timestamp - b.timestamp);
    return result.length > maxCapacity
        ? result.slice(result.length - maxCapacity)
        : result;
}

export function getCanvasContext(
    canvas: HTMLCanvasElement,
    width: number,
    height: number,
): CanvasRenderingContext2D | null {
    const dpr =
        typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
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

export interface PlotAreaRect {
    left: number;
    right: number;
    width: number;
}

export interface PlotLayout {
    left: number;
    right: number;
    width: number;
    canvasW: number;
}

export function computePlotLayout(
    canvas: HTMLCanvasElement | null,
    area: PlotAreaRect | undefined,
): PlotLayout {
    if (!canvas || !area) {
        return { left: 0, right: 0, width: 0, canvasW: 0 };
    }
    const canvasW = canvas.clientWidth || canvas.width;
    return {
        left: area.left,
        right: canvasW - area.right,
        width: area.width,
        canvasW,
    };
}

export interface DrawCursorOverlayOptions {
    cursorsEnabled: boolean;
    cursorA: number | null;
    cursorB: number | null;
    tags: ChartTag[];
    axes: ChartAxis[];
    history: Map<string, Point[]>;
}

export function renderCursorOverlay(
    chart: any,
    options: DrawCursorOverlayOptions,
): void {
    if (!options.cursorsEnabled) return;
    const scale = chart.scales.x;
    const area = chart.chartArea;
    if (!scale || !area) return;

    const min = Number(scale.min);
    const max = Number(scale.max);
    const cursorA = options.cursorA ?? Math.round(min + (max - min) * 0.25);
    const cursorB = options.cursorB ?? Math.round(min + (max - min) * 0.75);

    const xA = scale.getPixelForValue(cursorA);
    const xB = scale.getPixelForValue(cursorB);
    const ctx = chart.ctx as CanvasRenderingContext2D;
    ctx.save();

    const analogTags = options.tags.filter((t) => t.dataType !== 'Bool');
    const axes = options.axes;

    // Highlight band between A and B
    const leftX = Math.max(area.left, Math.min(xA, xB));
    const rightX = Math.min(area.right, Math.max(xA, xB));
    if (rightX > leftX) {
        ctx.fillStyle = CHART_THEME.cursorBandFill;
        ctx.fillRect(leftX, area.top, rightX - leftX, area.bottom - area.top);
    }

    // Draw Cursor A
    if (xA >= area.left && xA <= area.right) {
        ctx.beginPath();
        ctx.setLineDash([4, 2]);
        ctx.strokeStyle = CHART_THEME.cursorA;
        ctx.lineWidth = 1.5;
        ctx.moveTo(xA, area.top);
        ctx.lineTo(xA, area.bottom);
        ctx.stroke();

        // Top handle badge A
        ctx.setLineDash([]);
        ctx.fillStyle = CHART_THEME.cursorAFill;
        ctx.fillRect(
            xA - CHART_LAYOUT.cursorBadgeWidth / 2,
            area.top + 2,
            CHART_LAYOUT.cursorBadgeWidth,
            CHART_LAYOUT.cursorBadgeHeight,
        );
        ctx.fillStyle = CHART_THEME.white;
        ctx.font = CHART_FONTS.cursorBadge;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('A', xA, area.top + 9);

        // Intersection markers for active analog tags
        for (const tag of analogTags) {
            const valA = interpolateValue(
                options.history.get(tag.id) ?? [],
                cursorA,
            );
            if (valA !== null) {
                const axisIdx = Math.max(
                    0,
                    axes.findIndex((a) => a.name === tag.yAxis),
                );
                const yScale = chart.scales[`yAxis${axisIdx}`];
                if (yScale) {
                    const yPix = yScale.getPixelForValue(valA);
                    if (yPix >= area.top && yPix <= area.bottom) {
                        ctx.beginPath();
                        ctx.arc(
                            xA,
                            yPix,
                            CHART_LAYOUT.analogPointRadius,
                            0,
                            2 * Math.PI,
                        );
                        ctx.fillStyle = tag.color || CHART_THEME.cursorA;
                        ctx.fill();
                        ctx.strokeStyle = CHART_THEME.white;
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
        ctx.strokeStyle = CHART_THEME.cursorB;
        ctx.lineWidth = 1.5;
        ctx.moveTo(xB, area.top);
        ctx.lineTo(xB, area.bottom);
        ctx.stroke();

        // Top handle badge B
        ctx.setLineDash([]);
        ctx.fillStyle = CHART_THEME.cursorBFill;
        ctx.fillRect(
            xB - CHART_LAYOUT.cursorBadgeWidth / 2,
            area.top + 2,
            CHART_LAYOUT.cursorBadgeWidth,
            CHART_LAYOUT.cursorBadgeHeight,
        );
        ctx.fillStyle = CHART_THEME.white;
        ctx.font = CHART_FONTS.cursorBadge;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('B', xB, area.top + 9);

        // Intersection markers for active analog tags
        for (const tag of analogTags) {
            const valB = interpolateValue(
                options.history.get(tag.id) ?? [],
                cursorB,
            );
            if (valB !== null) {
                const axisIdx = Math.max(
                    0,
                    axes.findIndex((a) => a.name === tag.yAxis),
                );
                const yScale = chart.scales[`yAxis${axisIdx}`];
                if (yScale) {
                    const yPix = yScale.getPixelForValue(valB);
                    if (yPix >= area.top && yPix <= area.bottom) {
                        ctx.beginPath();
                        ctx.arc(
                            xB,
                            yPix,
                            CHART_LAYOUT.analogPointRadius,
                            0,
                            2 * Math.PI,
                        );
                        ctx.fillStyle = tag.color || CHART_THEME.cursorB;
                        ctx.fill();
                        ctx.strokeStyle = CHART_THEME.white;
                        ctx.lineWidth = 1.5;
                        ctx.stroke();
                    }
                }
            }
        }
    }

    ctx.restore();
}

export interface DrawBooleanBandOptions {
    start: number;
    end: number;
    layout: PlotLayout;
    boolTags: ChartTag[];
    history: Map<string, Point[]>;
    cursorsEnabled: boolean;
    cursorA: number | null;
    cursorB: number | null;
}

export function drawBooleanBand(
    ctx: CanvasRenderingContext2D,
    options: DrawBooleanBandOptions,
): void {
    const { start, end, layout, boolTags, history, cursorsEnabled, cursorA, cursorB } = options;
    const canvasW = layout.canvasW;
    const rowHeight = CHART_LAYOUT.boolRowHeight;
    const totalHeight = boolTags.length * rowHeight;

    ctx.clearRect(0, 0, canvasW, totalHeight);

    // Background
    ctx.fillStyle = CHART_THEME.boolBandBg;
    ctx.fillRect(0, 0, canvasW, totalHeight);

    const span = Math.max(1, end - start);
    const spanSeconds = span / 1000;
    const { stepMs, majorMs } = chooseXGridIntervals(spanSeconds);
    const firstGrid = Math.ceil(start / stepMs) * stepMs;

    boolTags.forEach((tag, index) => {
        const topY = index * rowHeight;

        // Bottom border for each row
        ctx.strokeStyle = CHART_THEME.borderMuted;
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
            ctx.font = CHART_FONTS.base;
            ctx.fillStyle = CHART_THEME.textMuted;
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
                ctx.strokeStyle = major
                    ? CHART_THEME.gridMajor
                    : CHART_THEME.gridMinor;
                ctx.lineWidth = major ? 1 : 0.8;
                ctx.setLineDash(major ? [] : [4, 4]);

                ctx.beginPath();
                ctx.moveTo(Math.round(x) + 0.5, topY);
                ctx.lineTo(Math.round(x) + 0.5, topY + rowHeight);
                ctx.stroke();
            }

            // Data high state blocks
            const points = history.get(tag.id) ?? [];
            ctx.fillStyle = tag.color || CHART_THEME.defaultSeries;
            ctx.setLineDash([]);

            const barTop = topY + CHART_LAYOUT.boolBarPaddingY;
            const barHeight = CHART_LAYOUT.boolBarHeight;

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
    if (cursorsEnabled && layout.width > 0) {
        ctx.save();
        ctx.beginPath();
        ctx.rect(layout.left, 0, layout.width, totalHeight);
        ctx.clip();

        if (cursorA !== null && cursorA >= start && cursorA <= end) {
            const xA = layout.left + ((cursorA - start) / span) * layout.width;
            ctx.beginPath();
            ctx.setLineDash([4, 2]);
            ctx.strokeStyle = CHART_THEME.cursorA;
            ctx.lineWidth = 1.5;
            ctx.moveTo(xA, 0);
            ctx.lineTo(xA, totalHeight);
            ctx.stroke();
        }

        if (cursorB !== null && cursorB >= start && cursorB <= end) {
            const xB = layout.left + ((cursorB - start) / span) * layout.width;
            ctx.beginPath();
            ctx.setLineDash([4, 2]);
            ctx.strokeStyle = CHART_THEME.cursorB;
            ctx.lineWidth = 1.5;
            ctx.moveTo(xB, 0);
            ctx.lineTo(xB, totalHeight);
            ctx.stroke();
        }
        ctx.restore();
    }
}

export interface DrawTimeAxisOptions {
    start: number;
    end: number;
    layout: PlotLayout;
    cursorsEnabled: boolean;
    cursorA: number | null;
    cursorB: number | null;
}

export function drawTimeAxis(
    ctx: CanvasRenderingContext2D,
    options: DrawTimeAxisOptions,
): void {
    const { start, end, layout, cursorsEnabled, cursorA, cursorB } = options;
    const canvasW = layout.canvasW;
    const height = CHART_LAYOUT.timeAxisHeight;

    ctx.clearRect(0, 0, canvasW, height);

    // Background
    ctx.fillStyle = CHART_THEME.chartBg;
    ctx.fillRect(0, 0, canvasW, height);

    // Top border line
    ctx.strokeStyle = CHART_THEME.gridMinor;
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

    ctx.font = CHART_FONTS.mono;
    ctx.fillStyle = CHART_THEME.textMuted;
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
    if (cursorsEnabled) {
        const badgeSize = CHART_LAYOUT.cursorTimeBadgeSize;
        if (cursorA !== null && cursorA >= start && cursorA <= end) {
            const xA = layout.left + ((cursorA - start) / span) * layout.width;
            ctx.fillStyle = CHART_THEME.cursorAFill;
            ctx.fillRect(xA - badgeSize / 2, 3, badgeSize, badgeSize);
            ctx.fillStyle = CHART_THEME.white;
            ctx.font = CHART_FONTS.cursorTimeBadge;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('A', xA, 11);
        }

        if (cursorB !== null && cursorB >= start && cursorB <= end) {
            const xB = layout.left + ((cursorB - start) / span) * layout.width;
            ctx.fillStyle = CHART_THEME.cursorBFill;
            ctx.fillRect(xB - badgeSize / 2, 3, badgeSize, badgeSize);
            ctx.fillStyle = CHART_THEME.white;
            ctx.font = CHART_FONTS.cursorTimeBadge;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('B', xB, 11);
        }
    }

    ctx.restore();
}
