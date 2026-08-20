import { state } from '../store';

export interface ShortcutOptions {
    hasModalOpen: () => boolean;
    toggleCursors: () => void;
    menuSaveSettings: () => void;
    menuLoadSettings: () => void;
    menuTrendWindow: () => void;
    openYAxes: () => void;
    openPlcTags: () => void;
    startSampling: () => void;
    pauseSampling: () => void;
    stopSampling: () => void;
}

export function handleKeydown(e: KeyboardEvent, opts: ShortcutOptions): void {
    const target = e.target as HTMLElement | null;
    const isInput =
        target &&
        (target.tagName === 'INPUT' ||
            target.tagName === 'TEXTAREA' ||
            target.tagName === 'SELECT' ||
            target.isContentEditable);

    const modalOpen = opts.hasModalOpen();

    // Ctrl + M / Cmd + M / Alt + M: Toggle measurement cursors
    if (
        (((e.ctrlKey || e.metaKey) && !e.altKey) ||
            (e.altKey && !e.ctrlKey && !e.metaKey)) &&
        (e.key === 'm' || e.key === 'M') &&
        !e.shiftKey
    ) {
        e.preventDefault();
        opts.toggleCursors();
        return;
    }

    // Ctrl + S / Cmd + S: Save settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 's' || e.key === 'S') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        e.preventDefault();
        opts.menuSaveSettings();
        return;
    }

    // Ctrl + L / Ctrl + O / Cmd + L / Cmd + O: Load settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 'l' || e.key === 'L' || e.key === 'o' || e.key === 'O') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        e.preventDefault();
        opts.menuLoadSettings();
        return;
    }

    // Ctrl + X / Cmd + X: X-axis settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 'x' || e.key === 'X') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        if (!isInput && !modalOpen) {
            e.preventDefault();
            opts.menuTrendWindow();
            return;
        }
    }

    // Ctrl + Y / Cmd + Y: Y-axis settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 'y' || e.key === 'Y') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        if (!isInput && !modalOpen) {
            e.preventDefault();
            opts.openYAxes();
            return;
        }
    }

    // Ctrl + T / Cmd + T: PLC / Tag settings
    if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === 't' || e.key === 'T') &&
        !e.shiftKey &&
        !e.altKey
    ) {
        if (!isInput && !modalOpen) {
            e.preventDefault();
            opts.openPlcTags();
            return;
        }
    }

    // Shift + F5: Stop sampling
    if (e.shiftKey && e.key === 'F5') {
        e.preventDefault();
        opts.stopSampling();
        return;
    }

    // F5: Start sampling (or resume if paused)
    if (
        !e.shiftKey &&
        !e.ctrlKey &&
        !e.altKey &&
        !e.metaKey &&
        e.key === 'F5'
    ) {
        e.preventDefault();
        if (!state.isSampling) {
            opts.startSampling();
        } else if (state.isPaused) {
            opts.pauseSampling();
        }
        return;
    }

    // Space: Toggle pause / resume (only when no modal is open and not typing inside an input)
    if (
        !e.ctrlKey &&
        !e.altKey &&
        !e.metaKey &&
        !e.shiftKey &&
        (e.key === ' ' || e.code === 'Space')
    ) {
        if (!isInput && !modalOpen && state.isSampling) {
            e.preventDefault();
            opts.pauseSampling();
            return;
        }
    }
}
