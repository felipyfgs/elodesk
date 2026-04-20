<script setup lang="ts">
import { VisXYContainer, VisLine, VisAxis } from '@unovis/vue'
import type { OverviewReport } from '~/types/reports'

const props = defineProps<{ report: OverviewReport | null }>()

const data = computed(() => {
  return (props.report?.volumeByDay ?? []).map(v => ({
    x: new Date(v.day).getTime(),
    y: v.total
  }))
})

const xAccessor = (d: { x: number }) => d.x
const yAccessor = (d: { y: number }) => d.y
</script>

<template>
  <UPageCard title="Volume" variant="subtle">
    <div class="h-64">
      <VisXYContainer :data="data" class="h-full w-full">
        <VisLine :x="xAccessor" :y="yAccessor" />
        <VisAxis type="x" :tick-format="(v: number) => new Date(v).toLocaleDateString()" />
        <VisAxis type="y" />
      </VisXYContainer>
    </div>
  </UPageCard>
</template>
