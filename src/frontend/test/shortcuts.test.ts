import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import App from '../src/App.vue';
import { i18n } from '../src/i18n';
import { state } from '../src/store';

describe('Keyboard Shortcuts in App.vue', () => {
    let wrapper: any;

    beforeEach(() => {
        state.isSampling = false;
        state.isPaused = false;
        state.settings.tags = [];
        state.settings.plcLinks = [];
        vi.clearAllMocks();
    });

    afterEach(() => {
        if (wrapper) {
            wrapper.unmount();
            wrapper = null;
        }
    });

    it('handles F5 to start sampling and Shift+F5 to stop sampling', async () => {
        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        // Trigger F5
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'F5' }));
        await flushPromises();
        expect(state.isSampling).toBe(true);

        // Trigger Shift + F5
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'F5', shiftKey: true }));
        await flushPromises();
        expect(state.isSampling).toBe(false);
    });

    it('handles Space to toggle pause/resume when sampling is active', async () => {
        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        state.isSampling = true;
        state.isPaused = false;

        // Space pauses
        window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', code: 'Space' }));
        await flushPromises();
        expect(state.isPaused).toBe(true);

        // Space resumes
        window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', code: 'Space' }));
        await flushPromises();
        expect(state.isPaused).toBe(false);
    });

    it('handles Ctrl+S and Ctrl+L for settings save and load', async () => {
        const savePlcTagSettingsMock = vi.fn().mockResolvedValue(undefined);
        const loadPlcTagSettingsMock = vi.fn().mockResolvedValue({ plcLinks: [], tags: [] });

        (window as any).go.backend.App.SavePlcTagSettings = savePlcTagSettingsMock;
        (window as any).go.backend.App.LoadPlcTagSettings = loadPlcTagSettingsMock;

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        // Ctrl + S
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 's', ctrlKey: true }));
        await flushPromises();
        expect(savePlcTagSettingsMock).toHaveBeenCalledTimes(1);

        // Ctrl + L
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'l', ctrlKey: true }));
        await flushPromises();
        expect(loadPlcTagSettingsMock).toHaveBeenCalledTimes(1);

        // Ctrl + O
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'o', ctrlKey: true }));
        await flushPromises();
        expect(loadPlcTagSettingsMock).toHaveBeenCalledTimes(2);
    });

    it('handles Ctrl+X and Ctrl+Y for X/Y axis settings dialogs', async () => {
        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        // Ctrl + Y opens YAxesDialog
        expect(wrapper.findComponent({ name: 'YAxesDialog' }).props('open')).toBe(false);
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'y', ctrlKey: true }));
        await flushPromises();
        expect(wrapper.findComponent({ name: 'YAxesDialog' }).props('open')).toBe(true);

        // Close YAxesDialog
        wrapper.findComponent({ name: 'YAxesDialog' }).vm.$emit('close');
        await flushPromises();
        expect(wrapper.findComponent({ name: 'YAxesDialog' }).props('open')).toBe(false);

        // Ctrl + X opens X-axis PromptDialog
        expect(wrapper.findComponent({ name: 'PromptDialog' }).props('open')).toBe(false);
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'x', ctrlKey: true }));
        await flushPromises();
        expect(wrapper.findComponent({ name: 'PromptDialog' }).props('open')).toBe(true);
    });

    it('handles Ctrl+M and Alt+M to toggle measurement cursors', async () => {
        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        expect(wrapper.findComponent({ name: 'MeasurementStrip' }).props('open')).toBe(false);

        // Ctrl + M opens measurement strip
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'm', ctrlKey: true }));
        await flushPromises();
        expect(wrapper.findComponent({ name: 'MeasurementStrip' }).props('open')).toBe(true);

        // Alt + M also closes measurement strip
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'm', altKey: true }));
        await flushPromises();
        expect(wrapper.findComponent({ name: 'MeasurementStrip' }).props('open')).toBe(false);
    });

    it('handles Ctrl+T to open PLC/Tag settings dialog', async () => {
        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        expect(wrapper.findComponent({ name: 'PlcTagsDialog' }).props('open')).toBe(false);
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 't', ctrlKey: true }));
        await flushPromises();
        expect(wrapper.findComponent({ name: 'PlcTagsDialog' }).props('open')).toBe(true);
    });
});
