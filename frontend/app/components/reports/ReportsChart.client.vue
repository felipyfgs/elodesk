<script setup lang="ts">
import { VisXYContainer, VisArea, VisLine, VisAxis, VisTooltip, VisCrosshair, VisGroupedBar } from '@unovis/vue'
import type { OverviewReport } from '~/types/reports'

const props = defineProps<{ report: OverviewReport | null }>()
const { t } = useI18n()

const volumeData = computed(() => {
  return (props.report?.volumeByDay ?? []).map(v => ({
    x: new Date(v.day).getTime(),
    y: v.total
  }))
})

const statusLabels = computed(() => {
  const breakdown = props.report?.statusBreakdown
  if (!breakdown) return []
  return Object.keys(breakdown)
})

const statusData = computed(() => {
  const breakdown = props.report?.statusBreakdown
  if (!breakdown || !Object.keys(breakdown).length) return []
  return Object.entries(breakdown).map(([status, count], i) => ({
    x: i,
    status,
    y: count
  }))
})

const xAccessor = (d: { x: number }) => d.x
const yAccessor = (d: { y: number }) => d.y

const statusX = (d: { x: number }) => d.x
const statusY = (d: { y: number }) => d.y
const statusFormat = (i: number) => statusLabels.value[i] ?? ''

const formatDate = (v: number) => new Date(v).toLocaleDateString()
</script>

<template>
  <div class="space-y-6">
    <UCard :ui="{ header: 'flex items-center justify-between' }">
      <template #header>
        <h3 class="font-semibold">
          {{ t('reports.volume') }}
        </h3>
      </template>

      <UEmpty
        v-if="!volumeData.length"
        icon="i-lucide-chart-no-axes-combined"
        :title="t('reports.noVolumeData')"
        variant="subtle"
        size="sm"
      />

      <div v-else class="h-72">
        <VisXYContainer :data="volumeData" class="h-full w-full" :padding="{ top: 8 }">
          <VisArea
            :x="xAccessor"
            :y="yAccessor"
            curve-type="basis"
            color="var(--ui-primary)"
            :opacity="0.15"
          />
          <VisLine
            :x="xAccessor"
            :y="yAccessor"
            curve-type="basis"
            color="var(--ui-primary)"
            :line-width="2"
          />
          <VisAxis
            type="x"
            :tick-format="formatDate"
            :num-ticks="6"
            :grid-line="false"
          />
          <VisAxis type="y" :grid-line="true" :tick-format="(v: number) => String(v)" />
          <VisCrosshair :template="(d: { x: number, y: number }) => `${formatDate(d.x)}<br/><b>${d.y}</b>`" />
          <VisTooltip />
        </VisXYContainer>
      </div>
    </UCard>

    <UCard v-if="statusData.length" :ui="{ header: 'flex items-center justify-between' }">
      <template #header>
        <h3 class="font-semibold">
          {{ t('reports.statusBreakdown') }}
        </h3>
      </template>

      <div class="h-48">
        <VisXYContainer :data="statusData" class="h-full w-full" :padding="{ top: 8 }">
          <VisGroupedBar
            :x="statusX"
            :y="statusY"
            :bar-width="32"
            color="var(--ui-primary)"
            rounded-corners
          />
          <VisAxis type="x" :tick-format="statusFormat" :grid-line="false" />
          <VisAxis type="y" :grid-line="true" />
          <VisTooltip />
        </VisXYContainer>
      </div>
    </UCard>
  </div>
</template>
