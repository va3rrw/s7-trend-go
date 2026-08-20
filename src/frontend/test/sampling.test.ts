import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import App from '../src/App.vue';
import { i18n } from '../src/i18n';
import { state } from '../src/store';

describe('Sampling Settings Sync in App.vue', () => {
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

    it('updates backend with StartPolling when tags change during active sampling', async () => {
        const startPollingMock = vi.fn().mockResolvedValue(undefined);
        const saveSettingsMock = vi.fn().mockResolvedValue(undefined);

        (window as any).go.backend.App.StartPolling = startPollingMock;
        (window as any).go.backend.App.SaveSettings = saveSettingsMock;

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        state.isSampling = true;

        // Emit save from PlcTagsDialog
        const plcTagsDialog = wrapper.findComponent({ name: 'PlcTagsDialog' });
        plcTagsDialog.vm.$emit('save');
        await flushPromises();

        expect(startPollingMock).toHaveBeenCalledTimes(1);
        expect(saveSettingsMock).not.toHaveBeenCalled();
    });

    it('updates backend with SaveSettings when tags change while sampling is idle', async () => {
        const startPollingMock = vi.fn().mockResolvedValue(undefined);
        const saveSettingsMock = vi.fn().mockResolvedValue(undefined);

        (window as any).go.backend.App.StartPolling = startPollingMock;
        (window as any).go.backend.App.SaveSettings = saveSettingsMock;

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        state.isSampling = false;

        // Emit save from PlcTagsDialog
        const plcTagsDialog = wrapper.findComponent({ name: 'PlcTagsDialog' });
        plcTagsDialog.vm.$emit('save');
        await flushPromises();

        expect(saveSettingsMock).toHaveBeenCalledTimes(1);
        expect(startPollingMock).not.toHaveBeenCalled();
    });
});
