/**
 * Centralized theme colors, typography, layout geometry, and scale thresholds
 * for the trend chart and canvas rendering.
 */

// Theme Colors
export const CHART_THEME = {
    chartBg: '#0C1118',
    chartBorder: '#101820',
    boolBandBg: '#101820',
    gridMajor: '#5B7182',
    gridMinor: '#31414E',
    gridAlt: 'rgba(49, 65, 78, 0.5)',
    textMuted: '#CBD5E1',
    borderMuted: '#536572',
    cursorA: '#38BDF8',
    cursorAFill: '#0284C7',
    cursorB: '#FBBF24',
    cursorBFill: '#D97706',
    cursorBandFill: 'rgba(56, 189, 248, 0.07)',
    defaultSeries: '#93C5FD',
    white: '#FFFFFF',
} as const;

// Geometry & Layout Constants
export const CHART_LAYOUT = {
    boolRowHeight: 24,
    boolBarPaddingY: 6,
    boolBarHeight: 12,
    timeAxisHeight: 22,
    cursorHitTolerance: 12,
    cursorHoverTolerance: 8,
    cursorBadgeWidth: 20,
    cursorBadgeHeight: 14,
    cursorTimeBadgeSize: 16,
    analogPointRadius: 4,
    analogHoverRadius: 2,
} as const;

// Time & Scale Limits / Factors
export const CHART_LIMITS = {
    minTimeWindowSec: 30,
    maxTimeWindowSec: 86400,
    defaultTimeWindowSec: 60,
    zoomInFactor: 0.8,
    zoomOutFactor: 1.25,
    maxHistoryCapacity: 50000,
    pruneThreshold: 55000,
    historyFetchDebounceMs: 40,
    historyFetchPadRatio: 0.1,
    minFetchSpanMs: 1000,
    yAxisPadRatio: 0.08,
    yAxisMinPad: 0.001,
} as const;

// Typography
export const CHART_FONTS = {
    base: '11px "Segoe UI", -apple-system, BlinkMacSystemFont, Roboto, sans-serif',
    mono: '10px ui-monospace, "Cascadia Mono", "Segoe UI Mono", Consolas, monospace',
    cursorBadge: 'bold 10px sans-serif',
    cursorTimeBadge: 'bold 9px sans-serif',
} as const;

// Time grid steps (in seconds)
export const NICE_STEPS_SEC = [
    1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600,
] as const;
