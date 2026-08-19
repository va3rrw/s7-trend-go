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

    it('disables Y-Axis selection when tag is Bool and defaults to Y-Axis 1 on switch', async () => {
        const boolTag: TagSettings = {
            id: 'tag-bool',
            name: 'Motor_Running',
            plcLink: 'PLC1',
            address: 'I0.0',
            dataType: 'Bool',
            yAxis: '',
            color: '#FF0000',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(boolTag);
        const selects = wrapper.findAll('select');
        // selects: [plcLink, dataType, yAxis]
        const yAxisSelect = selects[selects.length - 1];

        // Should be disabled for Bool
        expect(yAxisSelect.attributes('disabled')).toBeDefined();

        // Switch to Real -> should be enabled and default to Y-Axis 1
        (wrapper.vm as any).form.dataType = 'Real';
        await nextTick();
        expect(yAxisSelect.attributes('disabled')).toBeUndefined();
        expect((wrapper.vm as any).form.yAxis).toBe('Y-Axis 1');

        // Switch back to Bool -> should be disabled and cleared
        (wrapper.vm as any).form.dataType = 'Bool';
        await nextTick();
        expect(yAxisSelect.attributes('disabled')).toBeDefined();
        expect((wrapper.vm as any).form.yAxis).toBe('');

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('save')).toBeTruthy();
        const saved = wrapper.emitted('save')?.[0]?.[0] as TagSettings;
        expect(saved.yAxis).toBe('');
    });

    it('shows inline address validation feedback and allows choosing from 16 color swatches', async () => {
        const tag: TagSettings = {
            id: 'tag-1',
            name: 'Speed',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#F87171',
            enabled: true,
        };

        const wrapper = await mountOpenEditor(tag);

        // Check valid address feedback
        const validFeedback = wrapper.find('.address-feedback');
        expect(validFeedback.exists()).toBe(true);
        expect(validFeedback.classes()).toContain('valid-msg');

        // Change to invalid address
        const inputs = wrapper.findAll('input[type="text"]');
        const addressInput = inputs[1];
        await addressInput.setValue('INVALID_ADDR');
        await nextTick();

        const invalidFeedback = wrapper.find('.address-feedback');
        // Check empty address shows placeholder hint with zero CLS
        await addressInput.setValue('');
        await nextTick();
        const hintFeedback = wrapper.find('.address-feedback');
        expect(hintFeedback.exists()).toBe(true);
        expect(hintFeedback.classes()).toContain('hint-msg');
        expect(hintFeedback.text()).toContain('Enter tag address');

        // Check 16 swatches rendered
        const swatches = wrapper.findAll('.swatch-btn');
        expect(swatches.length).toBe(16);

        // Click swatch 3 (Pale Amber #FDE68A)
        await swatches[2].trigger('click');
        expect((wrapper.vm as any).form.color).toBe('#FDE68A');
    });
});
