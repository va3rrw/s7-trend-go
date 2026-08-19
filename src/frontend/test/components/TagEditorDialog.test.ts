import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import TagEditorDialog from '../../src/components/TagEditorDialog.vue';
import type { TagSettings } from '../../src/types';
import { i18n } from '../../src/i18n';
import { nextTick } from 'vue';

describe('TagEditorDialog.vue', () => {
    async function mountOpenEditor(tag: TagSettings | null) {
        const wrapper = mount(TagEditorDialog, {
            props: {
                open: true,
                tag,
                index: 0,
            },
            global: {
                plugins: [i18n],
            },
        });
        await nextTick();
        return wrapper;
    }

    it('populates fields when opened with existing tag', async () => {
        const tag: TagSettings = {
            id: 'tag-1',
            name: 'Pressure_Sensor',
            plcLink: 'PLC1',
            address: 'DB2.DBD4',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        const nameInput = wrapper.find('input[type="text"]');
        expect((nameInput.element as HTMLInputElement).value).toBe('Pressure_Sensor');
    });

    it('emits save with modified tag on OK click', async () => {
        const tag: TagSettings = {
            id: 'tag-1',
            name: 'Tag_Original',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
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
        const tag: TagSettings = {
            id: 'tag-1',
            name: '',
            plcLink: 'PLC1',
            address: 'DB1.DBW0', // Invalid for Bool (missing bit)
            dataType: 'Bool',
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
        const tag: TagSettings = {
            id: 'tag-1',
            name: 'TestTag',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);
        const inputs = wrapper.findAll('input[type="text"]');
        const addressInput = inputs[1]; // 0 is name, 1 is address

        // Enter Bool address
        await addressInput.setValue('I0.1');
        await addressInput.trigger('input');
        expect((wrapper.vm as any).form.dataType).toBe('Bool');

        // Enter Byte address
        await addressInput.setValue('MB5');
        await addressInput.trigger('input');
        expect((wrapper.vm as any).form.dataType).toBe('Byte');

        // Enter Word/Int address
        await addressInput.setValue('DB1.DBW10');
        await addressInput.trigger('input');
        expect((wrapper.vm as any).form.dataType).toBe('Int');

        // Enter DBD/Real address
        await addressInput.setValue('DB1.DBD20');
        await addressInput.trigger('input');
        expect((wrapper.vm as any).form.dataType).toBe('Real');
    });

    it('prevents saving more than 16 tags on the same PLC link', async () => {
        // Create 16 existing tags on PLC1
        const existingTags: TagSettings[] = Array.from({ length: 16 }, (_, i) => ({
            id: `tag-${i}`,
            name: `Tag_${i}`,
            plcLink: 'PLC1',
            address: `DB1.DBD${i * 4}`,
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        }));

        // Attempt to create a 17th tag on PLC1 (index = -1 for new tag)
        const newTag: TagSettings = {
            id: 'tag-17',
            name: 'Overflow_Tag',
            plcLink: 'PLC1',
            address: 'DB1.DBD100',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = mount(TagEditorDialog, {
            props: {
                open: true,
                tag: newTag,
                index: -1,
                existingTags,
            },
            global: {
                plugins: [i18n],
            },
        });
        await nextTick();

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        // Should be blocked
        expect(wrapper.emitted('save')).toBeFalsy();

        // Switch to PLC2 (which has 0 tags)
        (wrapper.vm as any).form.plcLink = 'PLC2';
        await okBtn.trigger('click');

        // Should now succeed
        expect(wrapper.emitted('save')).toBeTruthy();
    });
});
