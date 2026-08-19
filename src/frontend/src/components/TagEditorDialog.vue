<template>
    <AppDialog
        :open="open"
        title="Tag Properties"
        width="460px"
        @close="emit('close')"
        @submit="save">
        <div class="form-grid">
            <div class="form-row">
                <label>{{ $t('dialog.name') }}</label>
                <input
                    v-model="form.name"
                    type="text" />
            </div>
            <div class="form-row">
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
                <input
                    v-model="form.address"
                    type="text"
                    @input="onAddressInput" />
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
                <select v-model="form.yAxis">
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
                <input
                    v-model="form.color"
                    type="color" />
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
        }
    },
    { immediate: true },
);

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

    emit('save', tag);
}
</script>
