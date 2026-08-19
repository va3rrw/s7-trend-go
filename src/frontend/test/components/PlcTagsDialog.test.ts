import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import PlcTagsDialog from '../../src/components/PlcTagsDialog.vue';
import { state, backend } from '../../src/store';
import { defaultSettings } from '../../src/types';
import { i18n } from '../../src/i18n';

describe('PlcTagsDialog.vue', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
        state.settings.tags = [
            {
                id: 'tag-1',
                name: 'Speed_PLC1',
                plcLink: 'PLC1',
                address: 'DB1.DBD0',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                color: '#FF0000',
                enabled: true,
            },
            {
                id: 'tag-2',
                name: 'Temp_PLC2',
                plcLink: 'PLC2',
                address: 'DB2.DBD0',
                dataType: 'Real',
                yAxis: 'Y-Axis 1',
                color: '#00FF00',
                enabled: true,
            },
        ];
    });

    async function mountOpenPlcTagsDialog() {
        const wrapper = mount(PlcTagsDialog, {
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

    it('renders PLC links on the left and PLC1 tags on the right by default', async () => {
        const wrapper = await mountOpenPlcTagsDialog();
        const plcRows = wrapper.findAll('.plc-panel table.datagrid tbody tr');
        expect(plcRows.length).toBe(1);

        const tagRows = wrapper.findAll('.tags-panel table.datagrid tbody tr');
        expect(tagRows.length).toBe(1);
        expect(tagRows[0].text()).toContain('Speed_PLC1');
    });

    it('switches tags list when clicking another PLC in the left panel', async () => {
        state.settings.plcLinks = [
            { name: 'PLC1', ipAddress: '192.168.0.1', rack: 0, slot: 1, isConnected: false },
            { name: 'PLC2', ipAddress: '192.168.0.2', rack: 0, slot: 1, isConnected: false },
        ];
        const wrapper = await mountOpenPlcTagsDialog();
        const plcRows = wrapper.findAll('.plc-panel table.datagrid tbody tr');
        expect(plcRows.length).toBe(2);

        // Click PLC2 (index 1)
        await plcRows[1].trigger('click');

        const tagRows = wrapper.findAll('.tags-panel table.datagrid tbody tr');
        expect(tagRows.length).toBe(1);
        expect(tagRows[0].text()).toContain('Temp_PLC2');
    });

    it('adds a tag to the currently selected PLC', async () => {
        const wrapper = await mountOpenPlcTagsDialog();
        const addBtn = wrapper.findAll('.tags-panel .panel-footer button')[0];
        await addBtn.trigger('click');

        const tagRows = wrapper.findAll('.tags-panel table.datagrid tbody tr');
        expect(tagRows.length).toBe(2);
    });

    it('removes a selected tag', async () => {
        const wrapper = await mountOpenPlcTagsDialog();
        const tagRow = wrapper.find('.tags-panel table.datagrid tbody tr');
        await tagRow.trigger('click');

        const removeBtn = wrapper.findAll('.tags-panel .panel-footer button')[2];
        await removeBtn.trigger('click');

        const tagRows = wrapper.findAll('.tags-panel table.datagrid tbody tr');
        // Empty state row is displayed
        expect(tagRows.length).toBe(1);
        expect(tagRows[0].classes()).not.toContain('selected');
    });

    it('tests connection for active PLC', async () => {
        const testConnSpy = vi.fn().mockResolvedValue(undefined);
        backend().TestConnection = testConnSpy;

        const wrapper = await mountOpenPlcTagsDialog();
        const testBtn = wrapper.findAll('.plc-panel .panel-footer button')[2];
        await testBtn.trigger('click');

        expect(testConnSpy).toHaveBeenCalledWith(
            expect.objectContaining({ name: 'PLC1' }),
        );
    });

    it('disables the Add button when the selected PLC reaches 16 tags', async () => {
        state.settings.plcLinks = [
            { name: 'PLC1', ipAddress: '192.168.0.1', rack: 0, slot: 1, isConnected: false },
            { name: 'PLC2', ipAddress: '192.168.0.2', rack: 0, slot: 1, isConnected: false },
        ];
        // Configure 16 tags for PLC1
        state.settings.tags = Array.from({ length: 16 }, (_, i) => ({
            id: `tag-${i}`,
            name: `Tag_${i}`,
            plcLink: 'PLC1',
            address: `DB1.DBD${i * 4}`,
            dataType: 'Real' as const,
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        }));

        const wrapper = await mountOpenPlcTagsDialog();
        const addBtn = wrapper.findAll('.tags-panel .panel-footer button')[0];
        expect(addBtn.attributes('disabled')).toBeDefined();

        // Switch to PLC2 (0 tags) -> Add button should become enabled
        const plcRows = wrapper.findAll('.plc-panel table.datagrid tbody tr');
        await plcRows[1].trigger('click');
        expect(addBtn.attributes('disabled')).toBeUndefined();
    });

    it('adds and removes PLC links respecting min (1) and max (8) bounds', async () => {
        const wrapper = await mountOpenPlcTagsDialog();
        const addPlcBtn = wrapper.findAll('.plc-panel .panel-footer button')[0];
        const removePlcBtn = wrapper.findAll('.plc-panel .panel-footer button')[1];

        // Initially 1 PLC -> Remove button should be disabled
        expect(removePlcBtn.attributes('disabled')).toBeDefined();

        // Add 7 more to reach 8
        for (let i = 2; i <= 8; i++) {
            await addPlcBtn.trigger('click');
        }
        let plcRows = wrapper.findAll('.plc-panel table.datagrid tbody tr');
        expect(plcRows.length).toBe(8);
        expect(addPlcBtn.attributes('disabled')).toBeDefined();

        // Remove down to 1 PLC
        for (let i = 8; i > 1; i--) {
            await removePlcBtn.trigger('click');
        }
        plcRows = wrapper.findAll('.plc-panel table.datagrid tbody tr');
        expect(plcRows.length).toBe(1);
        expect(removePlcBtn.attributes('disabled')).toBeDefined();
    });

    it('removes all associated tags when a PLC is removed', async () => {
        state.settings.plcLinks = [
            { name: 'PLC1', ipAddress: '192.168.0.1', rack: 0, slot: 1, isConnected: false },
            { name: 'PLC2', ipAddress: '192.168.0.2', rack: 0, slot: 1, isConnected: false },
        ];
        state.settings.tags = [
            { id: 'tag-1', name: 'T1', plcLink: 'PLC1', address: 'DB1.DBD0', dataType: 'Real', yAxis: 'Y-Axis 1', color: '#FF0000', enabled: true },
            { id: 'tag-2', name: 'T2', plcLink: 'PLC2', address: 'DB2.DBD0', dataType: 'Real', yAxis: 'Y-Axis 1', color: '#00FF00', enabled: true },
            { id: 'tag-3', name: 'T3', plcLink: 'PLC2', address: 'DB2.DBD4', dataType: 'Real', yAxis: 'Y-Axis 1', color: '#0000FF', enabled: true },
        ];

        const wrapper = await mountOpenPlcTagsDialog();
        const plcRows = wrapper.findAll('.plc-panel table.datagrid tbody tr');

        // Select PLC2 (index 1) and remove it
        await plcRows[1].trigger('click');
        const removePlcBtn = wrapper.findAll('.plc-panel .panel-footer button')[1];
        await removePlcBtn.trigger('click');

        // Save dialog
        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        // Verify only PLC1 tags remain in state
        expect(state.settings.plcLinks.length).toBe(1);
        expect(state.settings.plcLinks[0].name).toBe('PLC1');
        expect(state.settings.tags.length).toBe(1);
        expect(state.settings.tags[0].name).toBe('T1');
        expect(state.settings.tags.find((t) => t.plcLink === 'PLC2')).toBeUndefined();
    });

    it('saves both PLC links and tags to store on OK click', async () => {
        const wrapper = await mountOpenPlcTagsDialog();
        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('save')).toBeTruthy();
        expect(wrapper.emitted('close')).toBeTruthy();
    });
});
