import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import PlcDialog from '../../src/components/PlcDialog.vue';
import { state, backend } from '../../src/store';
import { defaultSettings } from '../../src/types';
import { i18n } from '../../src/i18n';

describe('PlcDialog.vue', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
    });

    async function mountOpenPlcDialog() {
        const wrapper = mount(PlcDialog, {
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

    it('renders PLC links in the datagrid table', async () => {
        const wrapper = await mountOpenPlcDialog();
        const rows = wrapper.findAll('table.datagrid tbody tr');
        expect(rows.length).toBe(4);
    });

    it('selects a row when clicked', async () => {
        const wrapper = await mountOpenPlcDialog();
        const rows = wrapper.findAll('table.datagrid tbody tr');
        await rows[2].trigger('click');
        expect(rows[2].classes()).toContain('selected');
    });

    it('saves changes to store and emits save & close', async () => {
        const wrapper = await mountOpenPlcDialog();
        const inputs = wrapper.findAll('table.datagrid tbody tr:first-child input[type="text"]');
        if (inputs[1]) {
            await inputs[1].setValue('10.0.0.50');
        }

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('save')).toBeTruthy();
        expect(wrapper.emitted('close')).toBeTruthy();
        expect(state.settings.plcLinks[0].ipAddress).toBe('10.0.0.50');
    });

    it('tests connection via backend API', async () => {
        const testConnSpy = vi.fn().mockResolvedValue(undefined);
        backend().TestConnection = testConnSpy;

        const wrapper = await mountOpenPlcDialog();
        const testBtn = wrapper.findAll('button.btn-outline')[0];
        await testBtn.trigger('click');

        expect(testConnSpy).toHaveBeenCalled();
    });
});
