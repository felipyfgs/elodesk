<script setup lang="ts">
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t: _t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const enabled = ref(false)

async function load() {
  if (!auth.account?.id) return
  try {
    const res = await api<{ enabled: boolean }>(`/accounts/${auth.account.id}/reports/csat`)
    enabled.value = !!res?.enabled
  } catch { enabled.value = false }
}

onMounted(load)
</script>

<template>
  <UDashboardPanel id="reports-csat">
    <template #header>
      <UDashboardNavbar title="CSAT">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="space-y-6 max-w-6xl mx-auto w-full">
        <template v-if="enabled">
          <ReportsCsatScoreCard :score="null" />
          <ReportsCsatDistributionChart />
          <ReportsCsatResponsesTable />
        </template>
        <UPageCard v-else title="CSAT desabilitado" variant="subtle">
          <p class="text-sm text-muted">
            Habilite CSAT nas configurações da conta para começar a coletar avaliações.
          </p>
          <template #footer>
            <UButton :to="`/accounts/${auth.account?.id}/settings/profile`" variant="outline">
              Configurações
            </UButton>
          </template>
        </UPageCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
