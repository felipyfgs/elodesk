<script setup lang="ts">
interface Row {
  id: number
  displayId: number
  inboxId: number
  assigneeId?: number | null
  status: number
  createdAt: string
}

const props = defineProps<{ items: Row[] }>()
const { t: _t } = useI18n()

function exportCsv() {
  const rows = [['id', 'displayId', 'inboxId', 'assigneeId', 'status', 'createdAt']]
  for (const r of props.items) {
    rows.push([String(r.id), String(r.displayId), String(r.inboxId), String(r.assigneeId ?? ''), String(r.status), r.createdAt])
  }
  const csv = rows.map(r => r.join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `conversations-report-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <UButton
    icon="i-lucide-download"
    variant="outline"
    size="sm"
    @click="exportCsv"
  >
    Export CSV
  </UButton>
</template>
