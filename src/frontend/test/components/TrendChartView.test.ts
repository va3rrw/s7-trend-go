import { describe, it, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import TrendChartView from '../../src/components/TrendChartView.vue';

describe('TrendChartView.vue', () => {
    it('mounts without errors and exposes methods', () => {
        const wrapper = mount(TrendChartView, {
            props: {
                tags: [
                    {
                        id: 'tag-1',
                        name: 'Tag 1',
                        color: '#FF0000',
                        dataType: 'Real',
                        yAxis: 'Y-Axis 1',
                    },
                ],
                axes: [
                    { name: 'Y-Axis 1', minimum: 0, maximum: 100, autoScale: true },
                ],
                timeWindowSeconds: 60,
                interpolation: 'Line',
            },
        });

        expect(wrapper.find('canvas').exists()).toBe(true);
        expect(wrapper.vm.addDataPoint).toBeInstanceOf(Function);
        expect(wrapper.vm.setPaused).toBeInstanceOf(Function);
        expect(wrapper.vm.clear).toBeInstanceOf(Function);

        // Call exposed methods
        expect(() => {
            wrapper.vm.addDataPoint('tag-1', new Date().toISOString(), 25.5);
            wrapper.vm.setPaused(true);
            wrapper.vm.clear();
        }).not.toThrow();

        wrapper.unmount();
    });
});
