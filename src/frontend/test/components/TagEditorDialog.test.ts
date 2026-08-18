import { describe, it, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import TagEditorDialog from '../../src/components/TagEditorDialog.vue';
import { state } from '../../src/store';
import { defaultSettings } from '../../src/types';
import { i18n } from '../../src/i18n';

describe('TagEditorDialog.vue', () => {
    beforeEach(() => {
        state.settings = defaultSettings();
    });

    async function mountOpenEditor(tag: any) {
        const wrapper = mount(TagEditorDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: false,
                tag,
                index: 0,
            },
        });
        await wrapper.setProps({ open: true });
        return wrapper;
    }

    it('populates form with tag properties when open', async () => {
        const tag = {
            id: 'tag-1',
            name: 'Pressure_Sensor',
            plcLink: 'PLC1',
            address: 'DB2.DBD4',
            dataType: 'Real',
            stringLength: 0,
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        const nameInput = wrapper.find('input[type="text"]');
        expect((nameInput.element as HTMLInputElement).value).toBe('Pressure_Sensor');
    });

    it('shows string length field only when dataType is String', async () => {
        const tag = {
            id: 'tag-1',
            name: 'StringTag',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'String',
            stringLength: 25,
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        expect(wrapper.text()).toContain('String length');
    });

    it('emits save with modified tag on OK click', async () => {
        const tag = {
            id: 'tag-1',
            name: 'Tag_Original',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            stringLength: 0,
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        const nameInput = wrapper.find('input[type="text"]');
        await nameInput.setValue('Tag_Modified');

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('save')).toBeTruthy();
        const savedTag = wrapper.emitted('save')?.[0]?.[0] as any;
        expect(savedTag.name).toBe('Tag_Modified');
    });

    it('validates required name and bool address format', async () => {
        const tag = {
            id: 'tag-1',
            name: '',
            plcLink: 'PLC1',
            address: 'DB1.DBW0', // Invalid for Bool (missing bit)
            dataType: 'Bool',
            stringLength: 0,
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        // Should not save when name is empty
        expect(wrapper.emitted('save')).toBeFalsy();
    });

    it('auto-selects data type based on entered address', async () => {
        const tag = {
            id: 'tag-1',
            name: 'TestTag',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            stringLength: 0,
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        const inputs = wrapper.findAll('input[type="text"]');
        const addressInput = inputs[1]; // 0 is name, 1 is address
        const select = wrapper.find('select:not([disabled])'); // dataType is second select

        // Enter Bool address
        await addressInput.setValue('I0.1');
        await addressInput.trigger('input');
        expect(wrapper.vm.form.dataType).toBe('Bool');

        // Enter Byte address
        await addressInput.setValue('MB5');
        await addressInput.trigger('input');
        expect(wrapper.vm.form.dataType).toBe('Byte');

        // Enter Word/Int address
        await addressInput.setValue('DB1.DBW10');
        await addressInput.trigger('input');
        expect(wrapper.vm.form.dataType).toBe('Int');

        // Enter DBD/Real address
        await addressInput.setValue('DB1.DBD20');
        await addressInput.trigger('input');
        expect(wrapper.vm.form.dataType).toBe('Real');
    });
});

