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
                    type="text" />
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
            <div
                v-if="showStringLength"
                class="form-row">
                <label>{{ $t('dialog.string_length') }}</label>
                <input
                    v-model.number="form.stringLength"
                    type="number" />
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
                <label>Notification</label>
                <select v-model="form.notification">
                    <option
                        v-for="n in NOTIFICATIONS"
                        :key="n"
                        :value="n">
                        {{ n }}
                    </option>
                </select>
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
import { DATA_TYPES, NOTIFICATIONS, PALETTE } from '../types';
import { state, showMessage } from '../store';
import { useI18n } from 'vue-i18n';
import AppDialog from './AppDialog.vue';

const props = defineProps<{
    open: boolean;
    tag: TagSettings | null;
    /** Index in tags array */
    index: number;
}>();
const emit = defineEmits<{ close: []; save: [tag: TagSettings] }>();
const { t } = useI18n();

const form = ref<TagSettings>({
    id: '',
    name: '',
    plcLink: '',
    address: '',
    dataType: 'Real',
    stringLength: 20,
    yAxis: '',
    lowLimit: -10,
    highLimit: 10,
    color: PALETTE[0],
    enabled: true,
    notification: 'None',
});

watch(
    () => props.open,
    (val) => {
        if (val && props.tag) {
            form.value = { ...props.tag };
        }
    },
);

const showStringLength = computed(() => form.value.dataType === 'String');

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

    emit('save', tag);
}
</script>
