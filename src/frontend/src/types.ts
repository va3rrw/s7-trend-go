export interface PlcLinkSettings {
    name: string;
    ipAddress: string;
    rack: number;
    slot: number;
    isConnected: boolean;
}

export interface TagSettings {
    id: string;
    name: string;
    plcLink: string;
    address: string;
    dataType: string;
    yAxis: string;
    color: string;
    enabled: boolean;
}

export interface YAxisSettings {
    name: string;
    minimum: number;
    maximum: number;
    autoScale: boolean;
}

export interface AppSettings {
    pollIntervalMs: number;
    timeWindowSeconds: number;
    interpolation: string;
    plcLinks: PlcLinkSettings[];
    tags: TagSettings[];
    yAxes: YAxisSettings[];
}

export interface SampleRecord {
    timestamp: string;
    tagName: string;
    address: string;
    value: number;
}

export const MIN_PLC_LINKS = 1;
export const MAX_PLC_LINKS = 8;
export const MAX_TAGS_PER_PLC_LINK = 16;

export const DATA_TYPES = [
    'Bool',
    'Byte',
    'Word',
    'Int',
    'DWord',
    'DInt',
    'Real',
    'LReal',
] as const;

export const PALETTE = [
    '#93C5FD',
    '#FDBA74',
    '#86EFAC',
    '#FDA4AF',
    '#C4B5FD',
    '#67E8F9',
    '#FDE68A',
    '#D8B4FE',
    '#BEF264',
    '#F9A8D4',
] as const;

export function newId(): string {
    return (
        globalThis.crypto?.randomUUID?.() ??
        `${Date.now()}-${Math.random().toString(16).slice(2)}`
    );
}

export function defaultSettings(): AppSettings {
    return {
        pollIntervalMs: 100,
        timeWindowSeconds: 60,
        interpolation: 'Line',
        plcLinks: [
            {
                name: 'PLC1',
                ipAddress: '192.168.0.1',
                rack: 0,
                slot: 1,
                isConnected: false,
            },
        ],
        tags: [],
        yAxes: [1, 2, 3, 4].map((i) => ({
            name: `Y-Axis ${i}`,
            minimum: 0,
            maximum: 100,
            autoScale: true,
        })),
    };
}

export function inferDataType(address: string, currentType?: string): string | null {
    const addr = address.trim().toUpperCase();
    if (!addr) return null;

    // Bit / Bool address
    if (
        /^DB\d+\.DBX\d+\.[0-7]$/i.test(addr) ||
        /^[MIQ](?:X)?\d+\.[0-7]$/i.test(addr) ||
        /^DB\d+\.DBX\d+$/i.test(addr)
    ) {
        return 'Bool';
    }

    // Byte address (8-bit)
    if (
        /^DB\d+\.DBB\d+$/i.test(addr) ||
        /^[MIQ]B\d+$/i.test(addr)
    ) {
        return 'Byte';
    }

    // Word / Int address (16-bit)
    if (
        /^DB\d+\.DBW\d+$/i.test(addr) ||
        /^[MIQ]W\d+$/i.test(addr)
    ) {
        if (currentType === 'Word') return 'Word';
        return 'Int';
    }

    // Double Word / DInt / Real address (32-bit)
    if (
        /^DB\d+\.DBD\d+$/i.test(addr) ||
        /^[MIQ]D\d+$/i.test(addr)
    ) {
        if (currentType === 'DInt' || currentType === 'DWord') return currentType;
        return 'Real';
    }

    return null;
}
