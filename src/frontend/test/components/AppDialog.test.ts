import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import AppDialog from '../../src/components/AppDialog.vue';

describe('AppDialog.vue', () => {
    it('renders title and slot content when mounted', () => {
        const wrapper = mount(AppDialog, {
            props: {
                open: true,
                title: 'Test Dialog Title',
                width: '500px',
            },
            slots: {
                default: '<div class="test-body">Body Content</div>',
                footer: '<button class="test-footer-btn">Footer Button</button>',
            },
        });

        expect(wrapper.find('.modal-header').text()).toBe('Test Dialog Title');
        expect(wrapper.find('.test-body').text()).toBe('Body Content');
        expect(wrapper.find('.test-footer-btn').text()).toBe('Footer Button');
    });

    it('emits close event when cancel is triggered', async () => {
        const wrapper = mount(AppDialog, {
            props: {
                open: true,
                title: 'Test Dialog',
            },
        });

        await wrapper.find('dialog').trigger('cancel');
        expect(wrapper.emitted('close')).toBeTruthy();
    });

    it('emits submit event on Enter keydown in body element', async () => {
        const wrapper = mount(AppDialog, {
            props: {
                open: true,
                title: 'Test Dialog',
            },
            slots: {
                default: '<input type="text" class="input-field" />',
            },
        });

        const input = wrapper.find('.input-field');
        await input.trigger('keydown', { key: 'Enter' });
        expect(wrapper.emitted('submit')).toBeTruthy();
    });
});
