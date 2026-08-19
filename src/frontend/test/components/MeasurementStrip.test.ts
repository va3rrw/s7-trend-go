import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import MeasurementStrip from '../../src/components/MeasurementStrip.vue';
import type { CursorMeasurement } from '../../src/measurement';
import type { TagSettings } from '../../src/types';

describe('MeasurementStrip.vue', () => {
    const sampleTags: TagSettings[] = [
        {
            id: 'tag-1',
            name: 'Speed',
            address: 'DB1.DBD0',
            dataType: 'Real',
            stringLength: 0,
            color: '#38BDF8',
            yAxis: 'Y-Axis 1',
            plcLink: 'PLC1',
            enabled: true,
        },
        {
            id: 'tag-2',
            name: 'Pressure',
            address: 'DB1.DBD4',
            dataType: 'Real',
            stringLength: 0,
            color: '#F43F5E',
            yAxis: 'Y-Axis 1',
            plcLink: 'PLC1',
            enabled: false,
        },
    ];

    const sampleMeasurement: CursorMeasurement = {
        cursorA: 1000,
        cursorB: 5000,
        deltaTSec: 4.0,
        deltaTMs: 4000,
        tags: {
            'tag-1': {
                valA: 10.5,
                valB: 50.5,
                deltaY: 40.0,
                slope: 10.0,
                min: 10.0,
                max: 55.0,
                mean: 32.5,
                stdDev: 15.2,
                count: 5,
            },
        },
    };

    it('renders cursor timestamps, delta T, and tag statistics when open', () => {
        const wrapper = mount(MeasurementStrip, {
            props: {
                open: true,
                measurement: sampleMeasurement,
                tags: sampleTags,
            },
            global: {
                mocks: {
                    $t: (msg: string) => msg,
                },
            },
        });

        expect(wrapper.find('.measurement-panel').exists()).toBe(true);
        expect(wrapper.find('.delta-val').text()).toContain('+4.000 s');
        expect(wrapper.find('.delta-val').text()).toContain('4000 ms');

        // Only enabled tags should be rendered in table
        const rows = wrapper.findAll('tbody tr');
        expect(rows.length).toBe(1);
        expect(rows[0].text()).toContain('Speed');
        expect(rows[0].text()).toContain('10.5');
        expect(rows[0].text()).toContain('50.5');
        expect(rows[0].text()).toContain('+40');
        expect(rows[0].text()).toContain('+10.000 /s');
    });

    it('does not render when open is false', () => {
        const wrapper = mount(MeasurementStrip, {
            props: {
                open: false,
                measurement: sampleMeasurement,
                tags: sampleTags,
            },
            global: {
                mocks: {
                    $t: (msg: string) => msg,
                },
            },
        });

        expect(wrapper.find('.measurement-panel').exists()).toBe(false);
    });

    it('emits fitWindow and close events on button clicks', async () => {
        const wrapper = mount(MeasurementStrip, {
            props: {
                open: true,
                measurement: sampleMeasurement,
                tags: sampleTags,
            },
            global: {
                mocks: {
                    $t: (msg: string) => msg,
                },
            },
        });

        await wrapper.find('.btn-close-strip').trigger('click');
        expect(wrapper.emitted('close')).toBeTruthy();

        await wrapper.find('.btn-outline').trigger('click');
        expect(wrapper.emitted('fitWindow')).toBeTruthy();
    });
});
