import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import AppToolbar from '../../src/components/AppToolbar.vue';
import { i18n } from '../../src/i18n';
import { state } from '../../src/store';

describe('AppToolbar.vue', () => {
    it('renders toolbar buttons and emits action events', async () => {
        state.isSampling = false;
        state.isPaused = false;

        const wrapper = mount(AppToolbar, {
            global: {
                plugins: [i18n],
            },
            props: {
                cursorsEnabled: false,
            },
        });

        // Start button
        const startBtn = wrapper.find('button.start');
        await startBtn.trigger('click');
        expect(wrapper.emitted('startSampling')).toBeTruthy();

        // PLC Tag settings button
        const plcBtn = wrapper.findAll('button.tool-btn')[0];
        await plcBtn.trigger('click');
        expect(wrapper.emitted('openPlcTags')).toBeTruthy();
    });
});
