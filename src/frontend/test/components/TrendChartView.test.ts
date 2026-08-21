import { describe, it, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import TrendChartView from '../../src/components/TrendChartView.vue';
import { i18n } from '../../src/i18n';

describe('TrendChartView.vue', () => {
    it('mounts without errors and exposes methods', () => {
        const wrapper = mount(TrendChartView, {
            global: {
                plugins: [i18n],
            },
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

    it('shows paused badge overlay when sampling is paused and emits resume on button click', async () => {
        const wrapper = mount(TrendChartView, {
            global: {
                plugins: [i18n],
            },
            props: {
                tags: [],
                axes: [],
                timeWindowSeconds: 60,
                interpolation: 'Line',
                isSampling: true,
                isPaused: true,
            },
        });

        const badge = wrapper.find('#chartPausedBadge');
        expect(badge.exists()).toBe(true);

        const resumeBtn = wrapper.find('#btnResumeOverlay');
        expect(resumeBtn.exists()).toBe(true);

        await resumeBtn.trigger('click');
        expect(wrapper.emitted('resume')).toBeTruthy();

        // If not paused, badge should not exist
        await wrapper.setProps({ isPaused: false });
        expect(wrapper.find('#chartPausedBadge').exists()).toBe(false);

        wrapper.unmount();
    });
});
