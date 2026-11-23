import { state, backend, isWebServerClient, resetStats } from '../store';
import type { AppSettings } from '../types';
import { saveSettingsAndRestart } from './sampling';

export type TranslateFn = (key: string, args?: any[]) => string;
export type PromptFn = (
    title: string,
    text: string,
    defaultValue: string,
    options?: string[],
) => Promise<string | null>;
export type MessageFn = (title: string, message: string) => Promise<boolean>;
export type ConfirmFn = (title: string, message: string) => Promise<boolean>;

export interface ChartClearable {
    clear(): void;
}

/**
 * Apply loaded settings object to reactive application state and reset tag stats.
 */
export function applyLoadedSettings(loaded: AppSettings): void {
    resetStats();
    if (loaded.pollIntervalMs) {
        state.settings.pollIntervalMs = loaded.pollIntervalMs;
    }
    if (loaded.timeWindowSeconds) {
        state.settings.timeWindowSeconds = loaded.timeWindowSeconds;
    }
    if (loaded.interpolation) {
        state.settings.interpolation = loaded.interpolation;
    }
    if (loaded.plcLinks?.length) {
        state.settings.plcLinks = loaded.plcLinks;
    }
    state.settings.tags = loaded.tags ?? [];
    if (loaded.yAxes?.length) {
        state.settings.yAxes = loaded.yAxes;
    }
}

async function pickJsonFileText(): Promise<string | null> {
    return new Promise((resolve) => {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.json,application/json';
        let settled = false;
        const finish = (val: string | null) => {
            if (settled) return;
            settled = true;
            input.remove();
            resolve(val);
        };
        input.addEventListener('change', async () => {
            const file = input.files?.[0];
            if (!file) {
                finish(null);
                return;
            }
            try {
                finish(await file.text());
            } catch {
                finish(null);
            }
        });
        input.addEventListener('cancel', () => finish(null));
        input.style.display = 'none';
        document.body.appendChild(input);
        input.click();
    });
}

export async function menuLoadSettings(t: TranslateFn): Promise<void> {
    try {
        if (isWebServerClient()) {
            const raw = await pickJsonFileText();
            if (!raw) return;
            const loaded = await backend()?.ImportSettingsJSON?.(raw);
            if (loaded) {
                applyLoadedSettings(loaded);
                state.statusMessage = t('status.loaded_settings');
            }
            return;
        }
        const loaded = await backend()?.LoadSettingsFile(
            t('menu.load_settings'),
        );
        if (loaded) {
            applyLoadedSettings(loaded);
            await saveSettingsAndRestart();
            state.statusMessage = t('status.loaded_settings');
        }
    } catch (err) {
        state.statusMessage = t('status.error_loading_settings', [err]);
    }
}

export async function menuSaveSettings(t: TranslateFn): Promise<void> {
    try {
        if (isWebServerClient()) {
            const json = JSON.stringify(state.settings, null, 2);
            const blob = new Blob([json], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = 's7-trend-settings.json';
            document.body.appendChild(link);
            link.click();
            link.remove();
            URL.revokeObjectURL(url);
            state.statusMessage = t('status.saved_settings');
            return;
        }
        await backend()?.SaveSettingsFile(
            state.settings,
            t('menu.save_settings'),
        );
        state.statusMessage = t('status.saved_settings');
    } catch (err) {
        state.statusMessage = t('status.error_saving_settings', [err]);
    }
}

export async function menuTimer(
    t: TranslateFn,
    showPrompt: PromptFn,
    showMessage: MessageFn,
): Promise<void> {
    const value = await showPrompt(
        t('menu.set_timer_interval'),
        t('prompt.ms_range'),
        String(state.settings.pollIntervalMs),
    );
    if (value === null) return;
    const interval = Number(value);
    if (!Number.isInteger(interval) || interval < 10 || interval > 60000) {
        showMessage(
            t('menu.set_timer_interval'),
            t('prompt.invalid_interval', ['10', '60000']),
        );
        return;
    }
    state.settings.pollIntervalMs = interval;
    state.settings.tags = state.settings.tags.map((tag) =>
        tag.samplingIntervalMs && tag.samplingIntervalMs < interval
            ? { ...tag, samplingIntervalMs: interval }
            : tag,
    );
    await saveSettingsAndRestart();
    state.statusMessage = t('status.interval_set', [interval]);
}

export async function menuTrendWindow(
    t: TranslateFn,
    showPrompt: PromptFn,
    showMessage: MessageFn,
): Promise<void> {
    const value = await showPrompt(
        t('menu.x_settings'),
        t('prompt.s_range'),
        String(state.settings.timeWindowSeconds),
    );
    if (value === null) return;
    const seconds = Number(value);
    if (!Number.isInteger(seconds) || seconds < 30 || seconds > 86400) {
        showMessage(
            t('menu.x_settings'),
            t('prompt.invalid_window', ['30', '86400']),
        );
        return;
    }
    state.settings.timeWindowSeconds = seconds;
    await saveSettingsAndRestart();
    state.statusMessage = t('status.window_set', [seconds]);
}

export async function menuInterpolation(
    t: TranslateFn,
    showPrompt: PromptFn,
): Promise<void> {
    const value = await showPrompt(
        t('menu.samples_interpolation'),
        t('prompt.mode'),
        state.settings.interpolation,
        ['Line', 'Step'],
    );
    if (value === null) return;
    state.settings.interpolation = value;
    await saveSettingsAndRestart();
    state.statusMessage = t('status.interpolation_set', [value]);
}

export async function menuClear(
    t: TranslateFn,
    showConfirm: ConfirmFn,
    chartRef?: ChartClearable | null,
    liveValues?: Record<string, string> | { value: Record<string, string> },
): Promise<void> {
    if (
        !(await showConfirm(t('menu.clear_records'), t('prompt.clear_confirm')))
    )
        return;
    chartRef?.clear();
    resetStats();
    if (liveValues) {
        if ('value' in liveValues && typeof (liveValues as any).value === 'object') {
            (liveValues as any).value = {};
        } else {
            Object.keys(liveValues).forEach((k) => delete (liveValues as any)[k]);
        }
    }
    state.statusMessage = t('status.records_cleared');
}

export async function menuExport(t: TranslateFn): Promise<void> {
    try {
        if (isWebServerClient()) {
            const link = document.createElement('a');
            link.href = '/api/export/csv';
            link.download = 'trend.csv';
            document.body.appendChild(link);
            link.click();
            link.remove();
            state.statusMessage = t('status.exported_data');
            return;
        }
        await backend()?.ExportCSV(t('menu.export_csv'));
        state.statusMessage = t('status.exported_data');
    } catch (err) {
        state.statusMessage = t('status.error_exporting', [err]);
    }
}

export async function menuOpenStorageFolder(t: TranslateFn): Promise<void> {
    try {
        await backend()?.OpenStorageFolder();
        state.statusMessage = t('status.storage_folder_opened');
    } catch (err) {
        state.statusMessage = t('status.error_opening_storage_folder', [err]);
    }
}

export async function menuToggleWebServer(t: TranslateFn): Promise<void> {
    try {
        const status = await backend()?.ToggleWebServer?.();
        if (!status) return;
        state.webServer = status;
        state.statusMessage = status.running
            ? t('status.web_server_started', [status.url || `:${status.port}`])
            : t('status.web_server_stopped');
    } catch (err) {
        const refreshed = await backend()?.GetWebServerStatus?.().catch(() => null);
        if (refreshed) {
            state.webServer = refreshed;
        }
        state.statusMessage = t('status.web_server_error', [err]);
    }
}

export async function menuWebServerPort(
    t: TranslateFn,
    showPrompt: PromptFn,
    showMessage: MessageFn,
): Promise<void> {
    const currentPort = state.webServer.port || 8080;
    const raw = await showPrompt(
        t('menu.web_server_port'),
        t('prompt.web_server_port'),
        String(currentPort),
    );
    if (raw === null) return;
    const trimmed = raw.trim();
    const port = Number.parseInt(trimmed, 10);
    if (!Number.isFinite(port) || String(port) !== trimmed || port < 1024 || port > 65535) {
        await showMessage(t('menu.web_server_port'), t('prompt.invalid_port'));
        return;
    }
    try {
        const status = await backend()?.SetWebServerPort?.(port);
        if (!status) return;
        state.webServer = status;
        state.statusMessage = status.running
            ? t('status.web_server_started', [status.url || `:${status.port}`])
            : t('status.web_server_port_set', [status.port]);
    } catch (err) {
        const refreshed = await backend()?.GetWebServerStatus?.().catch(() => null);
        if (refreshed) {
            state.webServer = refreshed;
        }
        state.statusMessage = t('status.web_server_error', [err]);
    }
}
