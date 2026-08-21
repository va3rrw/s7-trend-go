import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import ExitConfirmDialog from '../../src/components/ExitConfirmDialog.vue';
import { i18n } from '../../src/i18n';

describe('ExitConfirmDialog.vue', () => {
    it('renders with application font and emits save, dontSave, and cancel events', async () => {
        const wrapper = mount(ExitConfirmDialog, {
            global: {
                plugins: [i18n],
            },
            props: {
                open: true,
                title: 'Save Settings',
                message: 'Settings have been modified. Do you want to save changes before exiting?',
            },
        });

        expect(wrapper.text()).toContain('Settings have been modified');
        expect(wrapper.text()).toContain('Save');
        expect(wrapper.text()).toContain("Don't Save");
        expect(wrapper.text()).toContain('Cancel');

        const cancelBtn = wrapper.find('.btn-cancel');
        const dontSaveBtn = wrapper.find('.btn-dont-save');
        const saveBtn = wrapper.find('.btn-save');

        expect(cancelBtn.exists()).toBe(true);
        expect(dontSaveBtn.exists()).toBe(true);
        expect(saveBtn.exists()).toBe(true);

        await cancelBtn.trigger('click');
        expect(wrapper.emitted('cancel')).toBeTruthy();

        await dontSaveBtn.trigger('click');
        expect(wrapper.emitted('dontSave')).toBeTruthy();

        await saveBtn.trigger('click');
        expect(wrapper.emitted('save')).toBeTruthy();

        wrapper.unmount();
    });
});
