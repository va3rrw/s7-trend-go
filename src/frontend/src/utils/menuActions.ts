import { state, backend, resetStats } from '../store';
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

export async function menuLoadSettings(t: TranslateFn): Promise<void> {
    try {
        const loaded = await backend()?.LoadSettingsFile(
            t('menu.load_settings'),
        );
        if (loaded) {
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
            await saveSettingsAndRestart();
            state.statusMessage = t('status.loaded_settings');
        }
    } catch (err) {
        state.statusMessage = t('status.error_loading_settings', [err]);
    }
}

export async function menuSaveSettings(t: TranslateFn): Promise<void> {
    try {
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
    liveValues?: Record<string, string>,
): Promise<void> {
    if (
        !(await showConfirm(t('menu.clear_records'), t('prompt.clear_confirm')))
    )
        return;
    chartRef?.clear();
    resetStats();
    if (liveValues) {
        Object.keys(liveValues).forEach((k) => delete liveValues[k]);
    }
    state.statusMessage = t('status.records_cleared');
}

export async function menuExport(t: TranslateFn): Promise<void> {
    try {
        await backend()?.ExportCSV(t('menu.export_csv'));
        state.statusMessage = t('status.exported_data');
    } catch (err) {
        state.statusMessage = t('status.error_exporting', [err]);
    }
}
