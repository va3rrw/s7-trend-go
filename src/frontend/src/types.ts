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
    stringLength: number;
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

export const DATA_TYPES = [
    'Bool',
    'Byte',
    'Word',
    'Int',
    'DWord',
    'DInt',
    'Real',
    'LReal',
    'String',
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
        plcLinks: [0, 1, 2, 3].map((i) => ({
            name: `PLC${i + 1}`,
            ipAddress: `192.168.0.${i + 1}`,
            rack: 0,
            slot: 1,
            isConnected: false,
        })),
        tags: [],
        yAxes: [1, 2, 3, 4].map((i) => ({
            name: `Y-Axis ${i}`,
            minimum: 0,
            maximum: 100,
            autoScale: true,
        })),
    };
}
