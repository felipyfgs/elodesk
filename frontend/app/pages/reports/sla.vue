<script setup lang="ts">
import SlaKpiGrid from '~/components/reports/sla/SlaKpiGrid.vue'
import SlaViolationsTable from '~/components/reports/sla/SlaViolationsTable.vue'
import SlaPolicyBreakdown from '~/components/reports/sla/SlaPolicyBreakdown.vue'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

interface SlaReport {
  total: number
  met: number
  breached: number
  byPolicy: Array<{ policyId: number, policyName: string, total: number, breached: number }>
}

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const report = ref<SlaReport | null>(null)

async function load() {
  if (!auth.account?.id) return
  try {
    report.value = await api<SlaReport>(`/accounts/${auth.account.id}/reports/sla`)
  } catch { report.value = null }
}

onMounted(load)
</script>

<template>
  <UDashboardPanel id="reports-sla">
    <template #header>
      <UDashboardNavbar title="SLA">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="space-y-6 max-w-5xl mx-auto w-full">
        <template v-if="report && report.total > 0">
          <SlaKpiGrid :report="report" />
          <SlaViolationsTable :items="report.byPolicy ?? []" />
          <SlaPolicyBreakdown :items="report.byPolicy ?? []" />
        </template>
        <UPageCard v-else :title="t('settings.sla.empty')" variant="subtle">
          <p class="text-sm text-muted">
            Crie sua primeira política SLA para começar a medir violações.
          </p>
          <template #footer>
            <UButton to="/settings/sla" variant="outline">
              {{ t('settings.sla.new') }}
            </UButton>
          </template>
        </UPageCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
