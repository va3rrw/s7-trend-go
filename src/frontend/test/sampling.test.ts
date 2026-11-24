import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import App from '../src/App.vue';
import { i18n } from '../src/i18n';
import { state, backend, resetStats, updateTagValue } from '../src/store';
import { defaultSettings } from '../src/types';

describe('Sampling Settings Sync in App.vue', () => {
    let wrapper: any;

    beforeEach(() => {
        state.settings = defaultSettings();
        state.isSampling = false;
        state.isPaused = false;
        state.settings.tags = [];
        state.settings.plcLinks = [];
        resetStats();
        vi.clearAllMocks();
    });

    afterEach(() => {
        if (wrapper) {
            wrapper.unmount();
            wrapper = null;
        }
    });

    it('updates backend with StartPolling when tags change during active sampling', async () => {
        const startPollingMock = vi.fn().mockResolvedValue(undefined);
        const saveSettingsMock = vi.fn().mockResolvedValue(undefined);

        backend()!.StartPolling = startPollingMock;
        backend()!.SaveSettings = saveSettingsMock;

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        state.isSampling = true;

        // Emit a settings change from the persistent PLC/tag tree.
        const plcTagTree = wrapper.findComponent({ name: 'PlcTagTree' });
        plcTagTree.vm.$emit('settingsChange');
        await flushPromises();

        expect(startPollingMock).toHaveBeenCalledTimes(1);
        expect(saveSettingsMock).not.toHaveBeenCalled();
    });

    it('updates backend with SaveSettings when tags change while sampling is idle', async () => {
        const startPollingMock = vi.fn().mockResolvedValue(undefined);
        const saveSettingsMock = vi.fn().mockResolvedValue(undefined);

        backend()!.StartPolling = startPollingMock;
        backend()!.SaveSettings = saveSettingsMock;

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        state.isSampling = false;

        // Emit a settings change from the persistent PLC/tag tree.
        const plcTagTree = wrapper.findComponent({ name: 'PlcTagTree' });
        plcTagTree.vm.$emit('settingsChange');
        await flushPromises();

        expect(saveSettingsMock).toHaveBeenCalledTimes(1);
        expect(startPollingMock).not.toHaveBeenCalled();
    });

    it('defers adding and polling a new inherited tag until TagEditorDialog is confirmed', async () => {
        const startPollingMock = vi.fn().mockResolvedValue(undefined);
        const existingTag = {
            id: 'tag-existing',
            name: 'ExistingTag',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FCA5A5',
            enabled: true,
            samplingIntervalMs: 0,
        };
        const initialSettings = {
            ...defaultSettings(),
            plcLinks: [
                {
                    name: 'PLC1',
                    ipAddress: '192.168.0.1',
                    rack: 0,
                    slot: 1,
                    isConnected: true,
                },
            ],
            tags: [existingTag],
        };

        backend()!.GetSettings = vi.fn().mockResolvedValue(initialSettings);
        backend()!.StartPolling = startPollingMock;

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        state.isSampling = true;
        updateTagValue(existingTag.id, '25', 25);

        // Click "Add tag" in the PLC/Tag tree
        await wrapper.find('.tree-add-tag-action').trigger('click');
        await flushPromises();

        // TagEditorDialog opens pre-populated with inherited values, but state/polling must NOT update yet
        const editor = wrapper.findAllComponents({ name: 'TagEditorDialog' }).at(-1)!;
        expect(editor.props('open')).toBe(true);
        expect(editor.props('tag').address).toBe('DB1.DBD0');
        expect(editor.props('tag').dataType).toBe('Real');
        expect(editor.props('tag').yAxis).toBe('Y-Axis 1');

        const draftId = editor.props('tag').id;
        expect(state.settings.tags).toHaveLength(1);
        expect(startPollingMock).not.toHaveBeenCalled();
        expect(state.sampledRange[draftId]).toBeUndefined();

        // Cancel dialog -> nothing added, no polling restart
        const cancelBtn = editor.findAll('button.btn-outline').find((b: any) =>
            b.text().includes('Cancel'),
        );
        await cancelBtn.trigger('click');
        await flushPromises();

        expect(editor.props('open')).toBe(false);
        expect(state.settings.tags).toHaveLength(1);
        expect(startPollingMock).not.toHaveBeenCalled();

        // Re-open Add Tag, update address, and save -> tag is committed & polling restarts once
        await wrapper.find('.tree-add-tag-action').trigger('click');
        await flushPromises();

        const newTagId = editor.props('tag').id;
        const addressInput = editor.findAll('input[type="text"]')[1];
        await addressInput.setValue('DB1.DBD4');
        await addressInput.trigger('input');

        const okBtn = editor.find('button.btn-primary');
        await okBtn.trigger('click');
        await flushPromises();

        expect(editor.props('open')).toBe(false);
        expect(state.settings.tags).toHaveLength(2);
        expect(state.settings.tags[1].id).toBe(newTagId);
        expect(state.settings.tags[1].address).toBe('DB1.DBD4');
        expect(startPollingMock).toHaveBeenCalledTimes(1);
        expect(state.sampledRange[newTagId]).toBeUndefined();

        // Only after a poll update for the new tag arrives should min/max start recording
        updateTagValue(newTagId, '12.5', 12.5);
        expect(state.sampledRange[newTagId]).toEqual({ min: 12.5, max: 12.5 });
    });

    it('resets sampledRange when an existing tag changes address, and preserves it when only name changes', async () => {
        const tag = {
            id: 'tag-edit-1',
            name: 'Flow',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FCA5A5',
            enabled: true,
            samplingIntervalMs: 0,
        };
        const initialSettings = {
            ...defaultSettings(),
            plcLinks: [
                {
                    name: 'PLC1',
                    ipAddress: '192.168.0.1',
                    rack: 0,
                    slot: 1,
                    isConnected: true,
                },
            ],
            tags: [tag],
        };
        backend()!.GetSettings = vi.fn().mockResolvedValue(initialSettings);

        wrapper = mount(App, {
            global: {
                plugins: [i18n],
            },
        });
        await flushPromises();

        updateTagValue(tag.id, '10', 10);
        updateTagValue(tag.id, '50', 50);
        expect(state.sampledRange[tag.id]).toEqual({ min: 10, max: 50 });

        // Rename only -> sampledRange preserved
        const grid = wrapper.findComponent({ name: 'TagsDataGrid' });
        grid.vm.$emit('editTag', state.settings.tags[0]);
        await flushPromises();

        const editor = wrapper.findAllComponents({ name: 'TagEditorDialog' }).at(-1)!;
        const nameInput = editor.findAll('input[type="text"]')[0];
        await nameInput.setValue('FlowRenamed');
        await editor.find('button.btn-primary').trigger('click');
        await flushPromises();

        expect(state.settings.tags[0].name).toBe('FlowRenamed');
        expect(state.sampledRange[tag.id]).toEqual({ min: 10, max: 50 });

        // Change address -> sampledRange reset
        grid.vm.$emit('editTag', state.settings.tags[0]);
        await flushPromises();

        const addressInput = editor.findAll('input[type="text"]')[1];
        await addressInput.setValue('DB1.DBD8');
        await addressInput.trigger('input');
        await editor.find('button.btn-primary').trigger('click');
        await flushPromises();

        expect(state.settings.tags[0].address).toBe('DB1.DBD8');
        expect(state.sampledRange[tag.id]).toBeUndefined();
    });
});
