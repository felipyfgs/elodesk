<script setup lang="ts">
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

defineProps<{ items: AuditLogEntry[], loading: boolean }>()

const { t } = useI18n()

function fmt(ts: string) {
  try {
    return new Date(ts).toLocaleString()
  } catch {
    return ts
  }
}
</script>

<template>
  <div class="border border-default rounded-lg overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-muted text-left">
        <tr>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.auditLogs.date') }}
          </th>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.auditLogs.action') }}
          </th>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.auditLogs.entity') }}
          </th>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.auditLogs.user') }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td colspan="4" class="px-4 py-6 text-center text-muted">
            …
          </td>
        </tr>
        <tr v-else-if="items.length === 0">
          <td colspan="4" class="px-4 py-6 text-center text-muted">
            {{ t('settings.auditLogs.empty') }}
          </td>
        </tr>
        <template v-for="entry in items" :key="entry.id">
          <UCollapsible>
            <template #default="{ open }">
              <tr class="border-t border-default cursor-pointer">
                <td class="px-4 py-2 text-muted text-xs">
                  {{ fmt(entry.createdAt) }}
                </td>
                <td class="px-4 py-2 font-mono text-xs">
                  {{ entry.action }}
                </td>
                <td class="px-4 py-2 text-muted">
                  {{ entry.entityType }} #{{ entry.entityId ?? '-' }}
                </td>
                <td class="px-4 py-2 text-muted text-xs flex items-center gap-2">
                  <span>{{ entry.userId ?? '-' }}</span>
                  <UIcon :name="open ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="text-xs ml-auto" />
                </td>
              </tr>
            </template>
            <template #content>
              <tr class="border-t border-default bg-muted/40">
                <td colspan="4" class="px-4 py-3">
                  <pre class="text-xs font-mono whitespace-pre-wrap">{{ entry.metadata || '{}' }}</pre>
                </td>
              </tr>
            </template>
          </UCollapsible>
        </template>
      </tbody>
    </table>
  </div>
</template>
