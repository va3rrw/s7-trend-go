import { describe, it, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import YAxesDialog from '../../src/components/YAxesDialog.vue';
import { state } from '../../src/store';
import { defaultSettings } from '../../src/types';
import { i18n } from '../../src/i18n';

describe('YAxesDialog.vue', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
    });

    async function mountOpenYAxesDialog() {
        const wrapper = mount(YAxesDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: false,
            },
        });
        await wrapper.setProps({ open: true });
        return wrapper;
    }

    it('renders 4 Y axes rows in datagrid', async () => {
        const wrapper = await mountOpenYAxesDialog();
        const rows = wrapper.findAll('table.datagrid tbody tr');
        expect(rows.length).toBe(4);
    });

    it('saves valid axis configuration and emits save & close', async () => {
        const wrapper = await mountOpenYAxesDialog();
        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('save')).toBeTruthy();
        expect(wrapper.emitted('close')).toBeTruthy();
    });

    it('validates axis min/max range', async () => {
        const wrapper = await mountOpenYAxesDialog();
        const numInputs = wrapper.findAll('table.datagrid tbody tr:first-child input[type="number"]');

        await numInputs[0].setValue(100);
        await numInputs[1].setValue(50);

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        // Should not emit save when validation fails
        expect(wrapper.emitted('save')).toBeFalsy();
    });
});
