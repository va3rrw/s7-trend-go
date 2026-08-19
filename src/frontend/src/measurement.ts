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

    const inRange = points.filter(
        (p) => p.timestamp >= tMin && p.timestamp <= tMax,
    );

    const values: number[] = inRange.map((p) => p.value);
    if (valA !== null) values.push(valA);
    if (valB !== null) values.push(valB);

    if (values.length === 0) {
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

    const min = Math.min(...values);
    const max = Math.max(...values);
    const sum = values.reduce((acc, v) => acc + v, 0);
    const mean = sum / values.length;
    const variance =
        values.reduce((acc, v) => acc + Math.pow(v - mean, 2), 0) /
        values.length;
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
        count: values.length,
    };
}
