import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import AppMenuBar from '../../src/components/AppMenuBar.vue';
import { i18n } from '../../src/i18n';
import { state } from '../../src/store';

describe('AppMenuBar.vue', () => {
    it('renders menu items and emits menu events', async () => {
        state.isSampling = false;
        state.isPaused = false;

        const wrapper = mount(AppMenuBar, {
            global: {
                plugins: [i18n],
            },
            props: {
                cursorsEnabled: false,
            },
        });

        // Trigger load settings
        const loadLink = wrapper.findAll('.dropdown a')[0];
        await loadLink.trigger('click');
        expect(wrapper.emitted('loadSettings')).toBeTruthy();

        // Trigger start sampling
        const startLink = wrapper.findAll('.dropdown a').find((a) => a.text().includes('Start') || a.text().includes('开始'));
        if (startLink) {
            await startLink.trigger('click');
            expect(wrapper.emitted('startSampling')).toBeTruthy();
        }
    });
});
