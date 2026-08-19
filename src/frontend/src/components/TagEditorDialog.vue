<template>
    <AppDialog
        :open="open"
        title="Tag Properties"
        width="480px"
        @close="emit('close')"
        @submit="save">
        <div class="form-grid">
            <div class="form-row">
                <label>{{ $t('dialog.name') }}</label>
                <input
                    v-model="form.name"
                    type="text" />
            </div>
            <div
                v-if="!hidePlc"
                class="form-row">
                <label>{{ $t('grid.plc') }}</label>
                <select v-model="form.plcLink">
                    <option
                        v-for="n in plcNames"
                        :key="n"
                        :value="n">
                        {{ n }}
                    </option>
                </select>
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.address') }}</label>
                <div class="field-control">
                    <input
                        v-model="form.address"
                        type="text"
                        :class="{
                            'border-invalid': addressStatus.isError,
                            'border-valid': addressStatus.isValid,
                        }"
                        @input="onAddressInput" />
                    <span
                        class="address-feedback"
                        :class="{
                            'valid-msg': addressStatus.isValid,
                            'invalid-msg': addressStatus.isError,
                            'hint-msg': addressStatus.isHint,
                        }">
                        {{ addressStatus.prefix }}{{ addressStatus.message }}
                    </span>
                </div>
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.data_type') }}</label>
                <select v-model="form.dataType">
                    <option
                        v-for="dt in DATA_TYPES"
                        :key="dt"
                        :value="dt">
                        {{ dt }}
                    </option>
                </select>
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.y_axis') }}</label>
                <select
                    v-model="form.yAxis"
                    :disabled="form.dataType === 'Bool'">
                    <option
                        v-if="form.dataType === 'Bool'"
                        value="">
                        -
                    </option>
                    <option
                        v-for="a in axisNames"
                        :key="a"
                        :value="a">
                        {{ a }}
                    </option>
                </select>
            </div>
            <div class="form-row">
                <label>{{ $t('dialog.color') }}</label>
                <div class="color-row">
                    <input
                        v-model="form.color"
                        type="color"
                        class="color-input"
                        title="Custom color" />
                    <div class="swatches-grid">
                        <button
                            v-for="c in PALETTE"
                            :key="c"
                            type="button"
                            class="swatch-btn"
                            :class="{ active: form.color?.toUpperCase() === c.toUpperCase() }"
                            :style="{ backgroundColor: c }"
                            :title="c"
                            @click="form.color = c" />
                    </div>
                </div>
            </div>
            <div class="form-row">
                <label />
                <label class="checkbox-label">
                    <input
                        v-model="form.enabled"
                        type="checkbox" />
                    Enabled
                </label>
            </div>
        </div>

        <template #footer>
            <div style="flex: 1" />
            <button
                class="btn btn-primary"
                @click="save">
                {{ $t('buttons.ok') }}
            </button>
            <button
                class="btn btn-outline"
                @click="emit('close')">
                {{ $t('buttons.cancel') }}
            </button>
        </template>
    </AppDialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import type { TagSettings } from '../types';
import { DATA_TYPES, PALETTE, inferDataType, MAX_TAGS_PER_PLC_LINK } from '../types';
import { state, showMessage } from '../store';
import { useI18n } from 'vue-i18n';
import AppDialog from './AppDialog.vue';

const props = defineProps<{
    open: boolean;
    tag: TagSettings | null;
    /** Index in tags array */
    index: number;
    /** Existing tags list to validate PLC link capacity against */
    existingTags?: TagSettings[];
    /** Whether to hide PLC link selector when locked/selected */
    hidePlc?: boolean;
}>();
const emit = defineEmits<{ close: []; save: [tag: TagSettings] }>();
const { t } = useI18n();

const form = ref<TagSettings>({
    id: '',
    name: '',
    plcLink: '',
    address: '',
    dataType: 'Real',
    yAxis: '',
    color: PALETTE[0],
    enabled: true,
});

watch(
    () => props.open,
    (val) => {
        if (val && props.tag) {
            form.value = { ...props.tag };
            if (form.value.dataType === 'Bool') {
                form.value.yAxis = '';
            }
        }
    },
    { immediate: true },
);

watch(
    () => form.value.dataType,
    (newType, oldType) => {
        if (newType === 'Bool') {
            form.value.yAxis = '';
        } else if (oldType === 'Bool' || !form.value.yAxis) {
            form.value.yAxis = axisNames.value[0] || 'Y-Axis 1';
        }
    },
);

const addressStatus = computed<{
    isValid: boolean;
    isError: boolean;
    isHint: boolean;
    prefix: string;
    message: string;
}>(() => {
    const addr = form.value.address.trim().toUpperCase();
    if (!addr) {
        return {
            isValid: false,
            isError: false,
            isHint: true,
            prefix: 'ℹ ',
            message: t('prompt.address_placeholder_hint'),
        };
    }

    if (form.value.dataType === 'Bool') {
        if (/^DB\d+\.DBX\d+\.[0-7]$/i.test(addr) || /^[MIQ](?:X)?\d+\.[0-7]$/i.test(addr)) {
            return {
                isValid: true,
                isError: false,
                isHint: false,
                prefix: '✓ ',
                message: t('prompt.valid_bit_address'),
            };
        }
        if (/^DB\d+\.DBX\d+$/i.test(addr)) {
            return {
                isValid: false,
                isError: true,
                isHint: false,
                prefix: '✕ ',
                message: t('prompt.missing_bit_number'),
            };
        }
        if (/^DB\d+\.(?:DBB|DBW|DBD)\d+$/i.test(addr) || /^[MIQ][BWD]\d+$/i.test(addr)) {
            return {
                isValid: false,
                isError: true,
                isHint: false,
                prefix: '✕ ',
                message: t('prompt.bool_requires_bit'),
            };
        }
        return {
            isValid: false,
            isError: true,
            isHint: false,
            prefix: '✕ ',
            message: t('prompt.invalid_bool_address'),
        };
    }

    // Non-bool (analog)
    if (/^DB\d+\.DBX\d+/i.test(addr) || /^[MIQ]X?\d+\.\d+$/i.test(addr)) {
        return {
            isValid: false,
            isError: true,
            isHint: false,
            prefix: '✕ ',
            message: t('prompt.bit_requires_bool'),
        };
    }
    if (/^DB\d+\.(?:DBB|DBW|DBD)\d+$/i.test(addr)) {
        return {
            isValid: true,
            isError: false,
            isHint: false,
            prefix: '✓ ',
            message: t('prompt.valid_db_address'),
        };
    }
    if (/^[MIQ][BWD]?\d+$/i.test(addr)) {
        return {
            isValid: true,
            isError: false,
            isHint: false,
            prefix: '✓ ',
            message: t('prompt.valid_io_address'),
        };
    }
    if (/^(?:T|TM)\d+$/i.test(addr)) {
        return {
            isValid: true,
            isError: false,
            isHint: false,
            prefix: '✓ ',
            message: t('prompt.valid_timer_address'),
        };
    }
    if (/^(?:C|Z|CT)\d+$/i.test(addr)) {
        return {
            isValid: true,
            isError: false,
            isHint: false,
            prefix: '✓ ',
            message: t('prompt.valid_counter_address'),
        };
    }

    return {
        isValid: false,
        isError: true,
        isHint: false,
        prefix: '✕ ',
        message: t('prompt.invalid_address_syntax'),
    };
});

function onAddressInput() {
    const inferred = inferDataType(form.value.address, form.value.dataType);
    if (inferred && inferred !== form.value.dataType) {
        form.value.dataType = inferred;
    }
}

const plcNames = computed(() => state.settings.plcLinks.map((p) => p.name));
const axisNames = computed(() => state.settings.yAxes.map((a) => a.name));

function save() {
    const tag = { ...form.value };
    tag.name = tag.name.trim();
    tag.address = tag.address.trim();

    if (!tag.name) {
        showMessage(t('dialog.tag_properties'), t('prompt.tag_name_required'));
        return;
    }
    if (
        tag.dataType === 'Bool' &&
        !/(?:^DB\d+\.DBX\d+\.[0-7]$|^[MIQ](?:X)?\d+\.[0-7]$)/i.test(tag.address)
    ) {
        showMessage(t('dialog.tag_properties'), t('prompt.invalid_bool_address'));
        return;
    }

    const existing = props.existingTags ?? state.settings.tags;
    const sameLinkCount = existing.filter(
        (t, idx) =>
            t.plcLink === tag.plcLink &&
            (props.index >= 0 ? idx !== props.index : t.id !== tag.id),
    ).length;

    if (sameLinkCount >= MAX_TAGS_PER_PLC_LINK) {
        showMessage(
            t('dialog.tag_properties'),
            t('prompt.max_tags_per_plc_exceeded', [tag.plcLink, MAX_TAGS_PER_PLC_LINK]),
        );
        return;
    }

    if (tag.dataType === 'Bool') {
        tag.yAxis = '';
    } else if (!tag.yAxis) {
        tag.yAxis = axisNames.value[0] || 'Y-Axis 1';
    }

    emit('save', tag);
}
</script>

<style scoped>
.field-control {
    display: flex;
    flex-direction: column;
    gap: 3px;
    flex: 1;
}

.border-valid {
    border-color: #10b981 !important;
}

.border-invalid {
    border-color: #ef4444 !important;
}

.address-feedback {
    font-size: 11px;
    line-height: 14px;
    height: 14px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-top: 1px;
}

.hint-msg {
    color: #64748b;
}

.valid-msg {
    color: #059669;
}

.invalid-msg {
    color: #dc2626;
}

.color-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
}

.color-input {
    width: 32px;
    height: 28px;
    padding: 1px 2px;
    border: 1px solid var(--border-color, #cbd5e1);
    border-radius: 4px;
    cursor: pointer;
    background: #ffffff;
    flex-shrink: 0;
}

.swatches-grid {
    display: grid;
    grid-template-columns: repeat(8, 1fr);
    gap: 4px;
    flex: 1;
}

.swatch-btn {
    width: 20px;
    height: 20px;
    border-radius: 3px;
    border: 1px solid rgba(0, 0, 0, 0.15);
    padding: 0;
    cursor: pointer;
    transition: transform 0.1s, box-shadow 0.1s;
    outline: none;
}

.swatch-btn:hover {
    transform: scale(1.18);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.swatch-btn.active {
    box-shadow: 0 0 0 2px #2563eb, 0 0 0 3px #ffffff;
    transform: scale(1.1);
}
</style>
