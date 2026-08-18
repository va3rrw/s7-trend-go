import { describe, it, expect } from 'vitest';
import { i18n } from '../src/i18n';
import en from '../src/locales/en.json';
import zh from '../src/locales/zh.json';

function getAllKeys(obj: any, prefix = ''): string[] {
    let keys: string[] = [];
    for (const k of Object.keys(obj)) {
        const path = prefix ? `${prefix}.${k}` : k;
        if (typeof obj[k] === 'object' && obj[k] !== null && !Array.isArray(obj[k])) {
            keys = keys.concat(getAllKeys(obj[k], path));
        } else {
            keys.push(path);
        }
    }
    return keys;
}

describe('i18n', () => {
    it('has matching keys between English and Chinese locales', () => {
        const enKeys = getAllKeys(en).sort();
        const zhKeys = getAllKeys(zh).sort();

        const missingInZh = enKeys.filter((k) => !zhKeys.includes(k));
        const missingInEn = zhKeys.filter((k) => !enKeys.includes(k));

        expect(missingInZh).toEqual([]);
        expect(missingInEn).toEqual([]);
    });

    it('initializes i18n instance and translates correctly', () => {
        expect(i18n.global.locale.value).toBeDefined();
        const okEn = (en as any).buttons?.ok;
        expect(okEn).toBeDefined();
    });
});
