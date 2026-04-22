<script setup lang="ts">
import EntityMetricsGrid from '~/components/reports/entity/EntityMetricsGrid.vue'
import EntityLineChart from '~/components/reports/entity/EntityLineChart.vue'
import type { EntityMetric } from '~/types/reports'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard', middleware: 'reports-entity' })

const route = useRoute()
const { t: _t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const entity = computed(() => String(route.params.entity))
const id = computed(() => Number(route.params.id))
const metric = ref<EntityMetric | null>(null)
const timeline = ref<Array<{ x: number, y: number }>>([])

async function load() {
  if (!auth.account?.id) return
  const items = await api<EntityMetric[]>(`/accounts/${auth.account.id}/reports/${entity.value}`)
  metric.value = items.find(m => m.entityId === id.value) ?? null
  // Timeline: reuse conversations report scoped by entity (best-effort)
  try {
    const convs = await api<{ payload?: Array<{ createdAt: string }> } | Array<{ createdAt: string }>>(
      `/accounts/${auth.account.id}/reports/conversations`,
      { query: entity.value === 'inboxes' ? { inbox_id: id.value } : { label_id: id.value } }
    )
    const rows = Array.isArray(convs) ? convs : (convs.payload ?? [])
    const byDay = new Map<string, number>()
    for (const r of rows) {
      const day = r.createdAt.slice(0, 10)
      byDay.set(day, (byDay.get(day) ?? 0) + 1)
    }
    timeline.value = Array.from(byDay.entries()).sort().map(([day, total]) => ({ x: new Date(day).getTime(), y: total }))
  } catch {
    timeline.value = []
  }
}

onMounted(load)
</script>

<template>
  <UDashboardPanel :id="`reports-${entity}-detail`">
    <template #header>
      <UDashboardNavbar :title="metric?.entityName ?? `${entity} #${id}`">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="space-y-6 max-w-6xl mx-auto w-full">
        <EntityMetricsGrid :metric="metric" />
        <ClientOnly>
          <EntityLineChart :data="timeline" />
        </ClientOnly>
      </div>
    </template>
  </UDashboardPanel>
</template>
