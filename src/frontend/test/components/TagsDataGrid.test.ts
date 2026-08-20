import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import TagsDataGrid from '../../src/components/TagsDataGrid.vue';
import { i18n } from '../../src/i18n';
import type { TagSettings } from '../../src/types';

describe('TagsDataGrid.vue', () => {
    const mockTags: TagSettings[] = [
        {
            id: 'tag-1',
            name: 'Speed',
            plcLink: 'PLC1',
            address: 'DB1.DBD0',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#FF0000',
            enabled: true,
        },
        {
            id: 'tag-2',
            name: 'Pressure',
            plcLink: 'PLC1',
            address: 'DB1.DBD4',
            dataType: 'Real',
            yAxis: 'Y-Axis 1',
            color: '#00FF00',
            enabled: true,
        },
    ];

    it('renders tags count, rows, and emits editTag on dblclick', async () => {
        const wrapper = mount(TagsDataGrid, {
            global: {
                plugins: [i18n],
            },
            props: {
                tags: mockTags,
                liveValues: { 'tag-1': '12.34', 'tag-2': '56.78' },
                sampledRange: {
                    'tag-1': { min: 10, max: 20 },
                    'tag-2': { min: 50, max: 60 },
                },
            },
        });

        expect(wrapper.text()).toContain('Speed');
        expect(wrapper.text()).toContain('12.34');

        // Double click tag cell
        const cell = wrapper.findAll('td.col-text')[0];
        await cell.trigger('dblclick');
        expect(wrapper.emitted('editTag')).toBeTruthy();
        expect(wrapper.emitted('editTag')![0][0]).toEqual(mockTags[0]);
    });

    it('filters rows by search query', async () => {
        const wrapper = mount(TagsDataGrid, {
            global: {
                plugins: [i18n],
            },
            props: {
                tags: mockTags,
                liveValues: {},
                sampledRange: {},
            },
        });

        const searchInput = wrapper.find('input.tags-search-input');
        await searchInput.setValue('Speed');

        expect(wrapper.text()).toContain('Speed');
        expect(wrapper.text()).not.toContain('Pressure');
    });
});
