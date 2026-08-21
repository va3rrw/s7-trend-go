export interface Point {
    timestamp: number;
    value: number;
}

export interface TagStats {
    valA: number | null;
    valB: number | null;
    deltaY: number | null;
    slope: number | null;
    min: number | null;
    max: number | null;
    mean: number | null;
    stdDev: number | null;
    count: number;
}

export interface CursorMeasurement {
    cursorA: number;
    cursorB: number;
    deltaTSec: number;
    deltaTMs: number;
    tags: Record<string, TagStats>;
}

export function interpolateValue(points: Point[], timestamp: number): number | null {
    if (!points || points.length === 0) return null;
    if (timestamp <= points[0].timestamp) return points[0].value;
    if (timestamp >= points[points.length - 1].timestamp)
        return points[points.length - 1].value;

    let low = 0;
    let high = points.length - 1;
    while (low <= high) {
        const mid = (low + high) >> 1;
        if (points[mid].timestamp === timestamp) {
            return points[mid].value;
        }
        if (points[mid].timestamp < timestamp) {
            low = mid + 1;
        } else {
            high = mid - 1;
        }
    }

    const p1 = points[high];
    const p2 = points[low];
    if (!p1 || !p2 || p2.timestamp === p1.timestamp) return p1?.value ?? null;

    const factor = (timestamp - p1.timestamp) / (p2.timestamp - p1.timestamp);
    return p1.value + factor * (p2.value - p1.value);
}

function findFirstIndex(points: Point[], tMin: number): number {
    let low = 0;
    let high = points.length - 1;
    let result = points.length;
    while (low <= high) {
        const mid = (low + high) >> 1;
        if (points[mid].timestamp >= tMin) {
            result = mid;
            high = mid - 1;
        } else {
            low = mid + 1;
        }
    }
    return result;
}

function findLastIndex(points: Point[], tMax: number): number {
    let low = 0;
    let high = points.length - 1;
    let result = -1;
    while (low <= high) {
        const mid = (low + high) >> 1;
        if (points[mid].timestamp <= tMax) {
            result = mid;
            low = mid + 1;
        } else {
            high = mid - 1;
        }
    }
    return result;
}

export function calculateTagStats(
    points: Point[],
    cursorA: number,
    cursorB: number,
): TagStats {
    const valA = interpolateValue(points, cursorA);
    const valB = interpolateValue(points, cursorB);

    const deltaY = valA !== null && valB !== null ? valB - valA : null;
    const deltaTSec = (cursorB - cursorA) / 1000;
    const slope =
        deltaY !== null && Math.abs(deltaTSec) > 0.00001
            ? deltaY / deltaTSec
            : null;

    const tMin = Math.min(cursorA, cursorB);
    const tMax = Math.max(cursorA, cursorB);

    let min: number | null = null;
    let max: number | null = null;
    let sum = 0;
    let count = 0;

    const includeValue = (v: number) => {
        if (min === null || v < min) min = v;
        if (max === null || v > max) max = v;
        sum += v;
        count++;
    };

    let lowIdx = 0;
    let highIdx = -1;
    if (points && points.length > 0) {
        lowIdx = findFirstIndex(points, tMin);
        highIdx = findLastIndex(points, tMax);
        for (let i = lowIdx; i <= highIdx; i++) {
            includeValue(points[i].value);
        }
    }

    if (valA !== null) includeValue(valA);
    if (valB !== null) includeValue(valB);

    if (count === 0) {
        return {
            valA,
            valB,
            deltaY,
            slope,
            min: null,
            max: null,
            mean: null,
            stdDev: null,
            count: 0,
        };
    }

    const mean = sum / count;
    let varianceSum = 0;

    if (points && points.length > 0 && lowIdx <= highIdx) {
        for (let i = lowIdx; i <= highIdx; i++) {
            const diff = points[i].value - mean;
            varianceSum += diff * diff;
        }
    }
    if (valA !== null) {
        const diff = valA - mean;
        varianceSum += diff * diff;
    }
    if (valB !== null) {
        const diff = valB - mean;
        varianceSum += diff * diff;
    }

    const variance = varianceSum / count;
    const stdDev = Math.sqrt(variance);

    return {
        valA,
        valB,
        deltaY,
        slope,
        min,
        max,
        mean,
        stdDev,
        count,
    };
}
