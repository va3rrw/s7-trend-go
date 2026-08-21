<template>
  <div class="chart-wrapper">
    <div class="chart-plot">
      <canvas ref="canvasRef" />
    </div>
    <canvas ref="boolBandRef" class="boolean-band hidden" />
    <canvas ref="timeAxisRef" class="time-axis-row" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import type { ChartTag, ChartAxis, CursorMeasurement } from '../chart'
import { TrendChart } from '../chart'

const props = defineProps<{
  tags: ChartTag[]
  axes: ChartAxis[]
  timeWindowSeconds: number
  interpolation: string
  cursorsEnabled?: boolean
}>()

const emit = defineEmits<{
  dragStart: []
  cursorChange: [meas: CursorMeasurement | null]
  windowChange: [seconds: number]
}>()

const canvasRef = ref<HTMLCanvasElement>()
const boolBandRef = ref<HTMLCanvasElement>()
const timeAxisRef = ref<HTMLCanvasElement>()
let chart: TrendChart | null = null

onMounted(() => {
  if (canvasRef.value && boolBandRef.value && timeAxisRef.value) {
    chart = new TrendChart(
      canvasRef.value,
      boolBandRef.value,
      timeAxisRef.value,
      () => emit('dragStart'),
    )
    chart.setTags(props.tags, props.axes)
    chart.setWindow(props.timeWindowSeconds)
    chart.setInterpolation(props.interpolation)
    chart.setOnCursorChange((meas) => emit('cursorChange', meas))
    chart.setOnWindowChange((seconds) => emit('windowChange', seconds))
    if (props.cursorsEnabled) {
      chart.setCursorsEnabled(true)
    }
  }
})

onBeforeUnmount(() => {
  chart?.destroy()
})

watch(
  () => [props.tags, props.axes] as const,
  ([tags, axes]) => {
    chart?.setTags(tags, axes)
  },
  { deep: true },
)

watch(
  () => props.timeWindowSeconds,
  (val) => {
    chart?.setWindow(val)
  },
)

watch(
  () => props.interpolation,
  (val) => {
    chart?.setInterpolation(val)
  },
)

watch(
  () => props.cursorsEnabled,
  (enabled) => {
    chart?.setCursorsEnabled(!!enabled)
  },
)

function addDataPoint(id: string, timestamp: string, value: number) {
  chart?.addDataPoint(id, timestamp, value)
}
function setPaused(paused: boolean) {
  chart?.setPaused(paused)
}
function clear() {
  chart?.clear()
}
function setCursorsEnabled(enabled: boolean) {
  chart?.setCursorsEnabled(enabled)
}
function fitCursorsToWindow() {
  chart?.fitCursorsToWindow()
}
function getMeasurements() {
  return chart?.getMeasurements() ?? null
}

defineExpose({
  addDataPoint,
  setPaused,
  clear,
  setCursorsEnabled,
  fitCursorsToWindow,
  getMeasurements,
})
</script>

