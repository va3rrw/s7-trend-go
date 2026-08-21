import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import AppStatusBar from '../../src/components/AppStatusBar.vue';
import { i18n } from '../../src/i18n';

describe('AppStatusBar.vue', () => {
    it('renders status items correctly', () => {
        const wrapper = mount(AppStatusBar, {
            global: {
                plugins: [i18n],
            },
            props: {
                statusMessage: 'Sampling active',
                timeWindowSeconds: 120,
                pollIntervalMs: 250,
                actualSamplingText: 'PLC1: 15ms',
            },
        });

        expect(wrapper.text()).toContain('Sampling active');
        expect(wrapper.text()).toContain('120 s');
        expect(wrapper.text()).toContain('250 ms');
        expect(wrapper.text()).toContain('Poll time:');
        expect(wrapper.text()).toContain('PLC1: 15ms');
    });
});
