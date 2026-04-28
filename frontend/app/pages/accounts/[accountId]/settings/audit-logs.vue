<script setup lang="ts">
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

interface AuditLogEntry {
  id: number
  accountId: number
  userId?: number | null
  action: string
  entityType?: string | null
  entityId?: number | null
  metadata?: string | null
  ipAddress?: string | null
  userAgent?: string | null
  createdAt: string
}

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const items = ref<AuditLogEntry[]>([])
const loading = ref(false)
const filters = reactive({ from: '', to: '', action: '', entityType: '' })

async function fetchPage() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<{ payload?: AuditLogEntry[] } | AuditLogEntry[]>(
      `/accounts/${auth.account.id}/audit_logs`,
      { query: { ...filters } }
    )
    items.value = Array.isArray(res) ? res : (res.payload ?? [])
  } finally {
    loading.value = false
  }
}

onMounted(fetchPage)
</script>

<template>
  <UPageCard :title="t('settings.auditLogs.title')" variant="subtle">
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton variant="outline" icon="i-lucide-refresh-cw" @click="fetchPage">
          {{ t('common.refresh') ?? 'Refresh' }}
        </UButton>
        <SettingsAuditLogsExportButton :items="items" />
      </div>
    </template>

    <div class="grid grid-cols-1 lg:grid-cols-[16rem_1fr] gap-6">
      <aside>
        <SettingsAuditLogsFilters
          v-model:from="filters.from"
          v-model:to="filters.to"
          v-model:action="filters.action"
          v-model:entity-type="filters.entityType"
        />
        <UButton class="mt-3 w-full" @click="fetchPage">
          {{ t('common.apply') ?? 'Apply' }}
        </UButton>
      </aside>
      <SettingsAuditLogsTable :items="items" :loading="loading" />
    </div>
  </UPageCard>
</template>
