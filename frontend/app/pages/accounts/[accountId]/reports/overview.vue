<script setup lang="ts">
import ReportsDateRangePicker from '~/components/reports/ReportsDateRangePicker.vue'
import ReportsPeriodSelect from '~/components/reports/ReportsPeriodSelect.vue'
import ReportsStats from '~/components/reports/ReportsStats.vue'
import ReportsChart from '~/components/reports/ReportsChart.client.vue'
import ReportsTopAgents from '~/components/reports/ReportsTopAgents.vue'
import type { OverviewReport, EntityMetric } from '~/types/reports'
import type { Range, Period } from '~/types'
import { useApi } from '~/composables/useApi'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useConversationsStore, type Conversation } from '~/stores/conversations'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const route = useRoute()
const aid = route.params.accountId as string
const inboxes = useInboxesStore()
const convs = useConversationsStore()

const report = ref<OverviewReport | null>(null)
const topAgents = ref<EntityMetric[]>([])
const period = ref<Period>('daily')
const range = ref<Range>({
  start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
  end: new Date()
})

async function load() {
  const q = { from: range.value.start.toISOString(), to: range.value.end.toISOString() }
  const [reportRes, agentsRes, inboxList, convRes] = await Promise.all([
    api<OverviewReport>(`/accounts/${aid}/reports/overview`, { query: q }),
    api<EntityMetric[]>(`/accounts/${aid}/reports/agents`, { query: q }),
    api<Inbox[]>(`/accounts/${aid}/inboxes`).catch(() => [] as Inbox[]),
    api<{ payload: Conversation[] }>(`/accounts/${aid}/conversations`).catch(() => ({ payload: [] as Conversation[] }))
  ])
  report.value = reportRes
  topAgents.value = agentsRes
  inboxes.setAll(inboxList)
  convs.setAll(convRes.payload ?? [])
}

watch([range, period], load, { deep: true })
onMounted(load)

const openCount = computed(() => convs.list.filter(c => c.status === 0).length)
const pendingCount = computed(() => convs.list.filter(c => c.status === 2).length)
const unreadCount = computed(() => convs.list.reduce((acc, c) => acc + (c.unreadCount ?? 0), 0))

const liveStats = computed(() => [
  { icon: 'i-lucide-inbox', title: t('home.stats.inboxes'), value: inboxes.list.length, to: `/accounts/${aid}/inboxes` },
  { icon: 'i-lucide-message-square', title: t('home.stats.open'), value: openCount.value, to: `/accounts/${aid}/conversations` },
  { icon: 'i-lucide-clock', title: t('home.stats.pending'), value: pendingCount.value, to: `/accounts/${aid}/conversations` },
  { icon: 'i-lucide-bell', title: t('home.stats.unread'), value: unreadCount.value, to: `/accounts/${aid}/conversations` }
])
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
        <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
          <UPageCard
            v-for="(stat, index) in liveStats"
            :key="index"
            :icon="stat.icon"
            :title="stat.title"
            :to="stat.to"
            variant="subtle"
            :ui="{
              container: 'gap-y-1.5',
              wrapper: 'items-start',
              leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25 flex-col',
              title: 'font-normal text-muted text-xs uppercase'
            }"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg hover:z-1"
          >
            <span class="text-2xl font-semibold text-highlighted">{{ stat.value }}</span>
          </UPageCard>
        </UPageGrid>

        <ReportsStats :report="report" />
        <ClientOnly>
          <ReportsChart :report="report" />
        </ClientOnly>
        <ReportsTopAgents :items="topAgents" />
      </div>
    </template>
  </UDashboardPanel>
</template>
