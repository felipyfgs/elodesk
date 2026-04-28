<script setup lang="ts">
import type { OverviewReport } from '~/types/reports'

const props = defineProps<{ report: OverviewReport | null }>()
const { t } = useI18n()

const stats = computed(() => [
  { icon: 'i-lucide-circle-dot', title: t('reports.open'), value: props.report?.openCount ?? 0 },
  { icon: 'i-lucide-circle-check', title: t('reports.resolved'), value: props.report?.resolvedCount ?? 0 },
  { icon: 'i-lucide-clock', title: t('reports.firstResponse'), value: props.report?.firstResponseAvgMinutes ? `${props.report.firstResponseAvgMinutes.toFixed(0)} min` : '—' },
  { icon: 'i-lucide-timer', title: t('reports.resolution'), value: props.report?.resolutionAvgMinutes ? `${props.report.resolutionAvgMinutes.toFixed(0)} min` : '—' }
])
</script>

<template>
  <UPageGrid class="lg:grid-cols-4">
    <UPageCard
      v-for="(stat, index) in stats"
      :key="index"
      :icon="stat.icon"
      :title="stat.title"
      variant="subtle"
      :ui="{
        container: 'gap-y-1.5',
        wrapper: 'items-start',
        leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25 flex-col',
        title: 'font-normal text-muted text-xs uppercase'
      }"
    >
      <span class="text-2xl font-semibold text-highlighted">{{ stat.value }}</span>
    </UPageCard>
  </UPageGrid>
</template>
