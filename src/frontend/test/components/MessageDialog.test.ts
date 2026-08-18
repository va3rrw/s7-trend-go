import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import MessageDialog from '../../src/components/MessageDialog.vue';
import { i18n } from '../../src/i18n';

describe('MessageDialog.vue', () => {
    it('renders message and OK button', () => {
        const wrapper = mount(MessageDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: true,
                title: 'Notification',
                message: 'Operation completed successfully.',
                showCancel: false,
            },
        });

        expect(wrapper.text()).toContain('Notification');
        expect(wrapper.text()).toContain('Operation completed successfully.');
        expect(wrapper.findAll('button').length).toBe(1); // Only OK button
    });

    it('renders Cancel button when showCancel is true', () => {
        const wrapper = mount(MessageDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: true,
                title: 'Confirm',
                message: 'Are you sure?',
                showCancel: true,
            },
        });

        const buttons = wrapper.findAll('button');
        expect(buttons.length).toBe(2); // Cancel and OK
    });

    it('emits close with true when OK is clicked', async () => {
        const wrapper = mount(MessageDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: true,
                title: 'Confirm',
                message: 'Are you sure?',
                showCancel: true,
            },
        });

        const okBtn = wrapper.find('button.btn-primary');
        await okBtn.trigger('click');

        expect(wrapper.emitted('close')).toBeTruthy();
        expect(wrapper.emitted('close')?.[0]).toEqual([true]);
    });

    it('emits close with false when Cancel is clicked', async () => {
        const wrapper = mount(MessageDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: true,
                title: 'Confirm',
                message: 'Are you sure?',
                showCancel: true,
            },
        });

        const cancelBtn = wrapper.find('button.btn-outline');
        await cancelBtn.trigger('click');

        expect(wrapper.emitted('close')).toBeTruthy();
        expect(wrapper.emitted('close')?.[0]).toEqual([false]);
    });
});
