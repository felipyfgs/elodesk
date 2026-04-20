<script setup lang="ts">
import ConversationsReportTable from '~/components/reports/conversations/ConversationsReportTable.vue'
import ConversationsReportFilters from '~/components/reports/conversations/ConversationsReportFilters.vue'
import ConversationsReportExport from '~/components/reports/conversations/ConversationsReportExport.vue'
import ReportsDateRangePicker from '~/components/reports/ReportsDateRangePicker.vue'
import type { Range } from '~/types/reports'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

interface Row {
  id: number
  displayId: number
  inboxId: number
  assigneeId?: number | null
  status: number
  createdAt: string
}

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const items = ref<Row[]>([])
const loading = ref(false)
const range = ref<Range>({
  start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
  end: new Date()
})
const filters = reactive({ inboxId: '', labelId: '' })

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<{ meta?: unknown, payload?: Row[] } | Row[]>(
      `/accounts/${auth.account.id}/reports/conversations`,
      {
        query: {
          from: range.value.start.toISOString(),
          to: range.value.end.toISOString(),
          inbox_id: filters.inboxId || undefined,
          label_id: filters.labelId || undefined
        }
      }
    )
    items.value = Array.isArray(res) ? res : (res.payload ?? [])
  } finally {
    loading.value = false
  }
}

watch([range, filters], load, { deep: true })
onMounted(load)
</script>

<template>
  <UDashboardPanel id="reports-conversations">
    <template #header>
      <UDashboardNavbar :title="t('reports.conversations')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <ConversationsReportExport :items="items" />
        </template>
      </UDashboardNavbar>
      <UDashboardToolbar>
        <ReportsDateRangePicker v-model="range" />
      </UDashboardToolbar>
    </template>
    <template #body>
      <div class="grid grid-cols-1 lg:grid-cols-[14rem_1fr] gap-6 max-w-7xl mx-auto w-full">
        <aside>
          <ConversationsReportFilters v-model:inbox-id="filters.inboxId" v-model:label-id="filters.labelId" />
        </aside>
        <ConversationsReportTable :items="items" :loading="loading" />
      </div>
    </template>
  </UDashboardPanel>
</template>
