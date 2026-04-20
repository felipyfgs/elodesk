<script setup lang="ts">
import type { OverviewReport } from '~/types/reports'

defineProps<{ report: OverviewReport | null }>()
const { t } = useI18n()
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
    <UPageCard :title="t('reports.open') ?? 'Open'" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ report?.openCount ?? 0 }}
      </p>
    </UPageCard>
    <UPageCard :title="t('reports.resolved') ?? 'Resolved'" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ report?.resolvedCount ?? 0 }}
      </p>
    </UPageCard>
    <UPageCard :title="t('reports.firstResponse') ?? 'First response'" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ report?.firstResponseAvgMinutes ? report.firstResponseAvgMinutes.toFixed(0) + ' min' : '—' }}
      </p>
    </UPageCard>
    <UPageCard :title="t('reports.resolution') ?? 'Resolution'" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ report?.resolutionAvgMinutes ? report.resolutionAvgMinutes.toFixed(0) + ' min' : '—' }}
      </p>
    </UPageCard>
  </div>
</template>
