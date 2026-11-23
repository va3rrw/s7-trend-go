import { describe, it, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AppMenuBar from '../../src/components/AppMenuBar.vue';
import { i18n } from '../../src/i18n';
import { state } from '../../src/store';

describe('AppMenuBar.vue', () => {
    beforeEach(() => {
        i18n.global.locale.value = 'en';
        delete (window as any).__S7_WEB_SERVER__;
        state.isSampling = false;
        state.isPaused = false;
        state.webServer = {
            enabled: false,
            running: false,
            host: '0.0.0.0',
            port: 8080,
            url: '',
            connectedClients: 0,
        };
    });

    it('renders desktop menu items including Web Server controls and emits events', async () => {
        const wrapper = mount(AppMenuBar, {
            global: {
                plugins: [i18n],
            },
            props: {
                cursorsEnabled: false,
            },
        });

        // Trigger load settings
        const loadLink = wrapper.findAll('.dropdown a')[0];
        await loadLink.trigger('click');
        expect(wrapper.emitted('loadSettings')).toBeTruthy();

        // Trigger start sampling
        const startLink = wrapper
            .findAll('.dropdown a')
            .find((a) => a.text().includes('START') || a.text().includes('Start'));
        if (startLink) {
            await startLink.trigger('click');
            expect(wrapper.emitted('startSampling')).toBeTruthy();
        }

        // Desktop mode shows Web Server toggle, port setting, and Open storage folder
        const toggleWebLink = wrapper
            .findAll('.dropdown a')
            .find((a) => a.text().includes('Web Server'));
        const portLink = wrapper
            .findAll('.dropdown a')
            .find((a) => a.text().includes('Web server port'));
        const storageLink = wrapper
            .findAll('.dropdown a')
            .find((a) => a.text().includes('Open storage folder'));

        expect(toggleWebLink).toBeDefined();
        expect(portLink).toBeDefined();
        expect(storageLink).toBeDefined();

        await toggleWebLink!.trigger('click');
        await portLink!.trigger('click');
        expect(wrapper.emitted('toggleWebServer')).toBeTruthy();
        expect(wrapper.emitted('setWebServerPort')).toBeTruthy();

        // Running web server shows checkmark and listening port
        state.webServer.running = true;
        state.webServer.port = 8080;
        await wrapper.vm.$nextTick();
        expect(toggleWebLink!.text()).toContain('✓');
        expect(toggleWebLink!.text()).toContain('(:8080)');
    });

    it('hides host-only Web Server controls and Open storage folder in remote web client mode', () => {
        (window as any).__S7_WEB_SERVER__ = true;

        const wrapper = mount(AppMenuBar, {
            global: {
                plugins: [i18n],
            },
            props: {
                cursorsEnabled: false,
            },
        });

        const linksText = wrapper.findAll('.dropdown a').map((a) => a.text());
        expect(linksText.some((txt) => txt.includes('Web Server'))).toBe(false);
        expect(linksText.some((txt) => txt.includes('Web server port'))).toBe(false);
        expect(linksText.some((txt) => txt.includes('Open storage folder'))).toBe(false);
        expect(linksText.some((txt) => txt.includes('About S7 Trend Go'))).toBe(true);
    });
});
