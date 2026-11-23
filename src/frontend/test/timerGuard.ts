import { afterAll } from 'vitest';

if (typeof window !== 'undefined') {
    const activeIntervals = new Set<ReturnType<typeof setInterval>>();
    const origSetInterval = window.setInterval.bind(window);
    const origClearInterval = window.clearInterval.bind(window);

    (window as any).setInterval = (
        handler: TimerHandler,
        timeout?: number,
        ...args: any[]
    ) => {
        const id = origSetInterval(handler as any, timeout, ...args);
        activeIntervals.add(id);
        return id;
    };

    (window as any).clearInterval = (id?: ReturnType<typeof setInterval>) => {
        if (id !== undefined) {
            activeIntervals.delete(id);
        }
        return origClearInterval(id);
    };

    afterAll(() => {
        for (const id of activeIntervals) {
            origClearInterval(id);
        }
        activeIntervals.clear();
    });
}
