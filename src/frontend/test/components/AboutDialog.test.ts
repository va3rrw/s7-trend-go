import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import AboutDialog from '../../src/components/AboutDialog.vue';
import { i18n } from '../../src/i18n';

describe('AboutDialog.vue', () => {
    it('renders version info and emits close when OK clicked', async () => {
        const wrapper = mount(AboutDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: true,
            },
        });

        expect(wrapper.text()).toContain('S7 Trend Go');
        expect(wrapper.text()).toContain('Ken Wang');

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('close')).toBeTruthy();
    });
});
