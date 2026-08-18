import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import PromptDialog from '../../src/components/PromptDialog.vue';
import { i18n } from '../../src/i18n';

describe('PromptDialog.vue', () => {
    it('renders input box when options are not provided', async () => {
        const wrapper = mount(PromptDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: false,
                title: 'Rename Tag',
                text: 'Enter new tag name:',
                defaultValue: 'Initial_Tag',
            },
        });
        await wrapper.setProps({ open: true });

        expect(wrapper.find('input#promptInput').exists()).toBe(true);
        expect(wrapper.find('select').exists()).toBe(false);

        const input = wrapper.find('input#promptInput');
        await input.setValue('Updated_Tag');

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('close')).toBeTruthy();
        expect(wrapper.emitted('close')?.[0]).toEqual(['Updated_Tag']);
    });

    it('renders select dropdown when options are provided', async () => {
        const wrapper = mount(PromptDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: false,
                title: 'Select Item',
                text: 'Choose an option:',
                defaultValue: 'Option B',
                options: ['Option A', 'Option B', 'Option C'],
            },
        });
        await wrapper.setProps({ open: true });

        expect(wrapper.find('select').exists()).toBe(true);
        expect(wrapper.find('input#promptInput').exists()).toBe(false);

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('close')).toBeTruthy();
        expect(wrapper.emitted('close')?.[0]).toEqual(['Option B']);
    });

    it('emits null on cancel', async () => {
        const wrapper = mount(PromptDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: false,
                title: 'Prompt',
                text: 'Text',
                defaultValue: 'Val',
            },
        });
        await wrapper.setProps({ open: true });

        const cancelBtn = wrapper.find('button.btn-outline');
        await cancelBtn.trigger('click');

        expect(wrapper.emitted('close')).toBeTruthy();
        expect(wrapper.emitted('close')?.[0]).toEqual([null]);
    });
});
