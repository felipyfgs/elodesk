<script setup lang="ts">
import ReportsDateRangePicker from '~/components/reports/ReportsDateRangePicker.vue'
import ReportsPeriodSelect from '~/components/reports/ReportsPeriodSelect.vue'
import ReportsStats from '~/components/reports/ReportsStats.vue'
import ReportsChart from '~/components/reports/ReportsChart.client.vue'
import ReportsTopAgents from '~/components/reports/ReportsTopAgents.vue'
import type { OverviewReport, Range, Period, EntityMetric } from '~/types/reports'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const report = ref<OverviewReport | null>(null)
const topAgents = ref<EntityMetric[]>([])
const period = ref<Period>('daily')
const range = ref<Range>({
  start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
  end: new Date()
})

async function load() {
  if (!auth.account?.id) return
  const q = { from: range.value.start.toISOString(), to: range.value.end.toISOString() }
  report.value = await api<OverviewReport>(`/accounts/${auth.account.id}/reports/overview`, { query: q })
  topAgents.value = await api<EntityMetric[]>(`/accounts/${auth.account.id}/reports/agents`, { query: q })
}

watch([range, period], load, { deep: true })
onMounted(load)
</script>

<template>
  <UDashboardPanel id="reports-overview">
    <template #header>
      <UDashboardNavbar :title="t('reports.overview')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
      <UDashboardToolbar>
        <div class="flex items-center gap-4 w-full">
          <ReportsDateRangePicker v-model="range" />
          <ReportsPeriodSelect v-model="period" />
        </div>
      </UDashboardToolbar>
    </template>
    <template #body>
      <div class="space-y-6 max-w-6xl mx-auto w-full">
        <ReportsStats :report="report" />
        <ClientOnly>
          <ReportsChart :report="report" />
        </ClientOnly>
        <ReportsTopAgents :items="topAgents" />
      </div>
    </template>
  </UDashboardPanel>
</template>
