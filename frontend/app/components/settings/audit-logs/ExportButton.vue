<script setup lang="ts">
interface AuditLogEntry {
  id: number
  action: string
  entityType?: string | null
  entityId?: number | null
  userId?: number | null
  createdAt: string
  metadata?: string | null
}

const props = defineProps<{ items: AuditLogEntry[] }>()
const { t } = useI18n()

function exportCsv() {
  const rows = [['id', 'createdAt', 'action', 'entityType', 'entityId', 'userId', 'metadata']]
  for (const e of props.items) {
    rows.push([
      String(e.id),
      e.createdAt,
      e.action,
      String(e.entityType ?? ''),
      String(e.entityId ?? ''),
      String(e.userId ?? ''),
      (e.metadata ?? '').replace(/"/g, '""')
    ])
  }
  const csv = rows.map(r => r.map(f => `"${f}"`).join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `audit-logs-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <UButton icon="i-lucide-download" variant="outline" @click="exportCsv">
    {{ t('settings.auditLogs.exportCsv') }}
  </UButton>
</template>
