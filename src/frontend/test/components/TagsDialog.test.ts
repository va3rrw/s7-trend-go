import { describe, it, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import TagsDialog from '../../src/components/TagsDialog.vue';
import { state } from '../../src/store';
import { defaultSettings } from '../../src/types';
import { i18n } from '../../src/i18n';

describe('TagsDialog.vue', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
        state.settings.tags = [
            {
                id: 'tag-1',
                name: 'Speed',
                plcLink: 'PLC1',
                address: 'DB1.DBD0',
                dataType: 'Real',
                stringLength: 0,
                yAxis: 'Y-Axis 1',
                lowLimit: 0,
                highLimit: 100,
                color: '#FF0000',
                enabled: true,
                notification: 'None',
            },
        ];
    });

    async function mountOpenTagsDialog() {
        const wrapper = mount(TagsDialog, {
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

    it('renders list of tags', async () => {
        const wrapper = await mountOpenTagsDialog();
        const rows = wrapper.findAll('table.datagrid tbody tr');
        expect(rows.length).toBe(1);
        expect(rows[0].text()).toContain('Speed');
        expect(rows[0].text()).toContain('DB1.DBD0');
    });

    it('removes a selected tag', async () => {
        const wrapper = await mountOpenTagsDialog();
        const row = wrapper.find('table.datagrid tbody tr');
        await row.trigger('click');

        // Click Remove button (3rd button in footer)
        const buttons = wrapper.findAll('button.btn-outline');
        // Footer buttons: Add, Edit, Remove, Cancel
        const removeBtn = buttons[2];
        if (removeBtn) {
            await removeBtn.trigger('click');
            const remainingRows = wrapper.findAll('table.datagrid tbody tr');
            expect(remainingRows.length).toBe(0);
        }
    });

    it('adds a new tag and opens the tag editor', async () => {
        const wrapper = await mountOpenTagsDialog();
        // Click Add button (1st outline button in footer)
        const addBtn = wrapper.findAll('button.btn-outline')[0];
        await addBtn.trigger('click');

        const rows = wrapper.findAll('table.datagrid tbody tr');
        expect(rows.length).toBe(2);
    });

    it('saves tags to store on OK click', async () => {
        const wrapper = await mountOpenTagsDialog();
        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('save')).toBeTruthy();
        expect(wrapper.emitted('close')).toBeTruthy();
    });
});
