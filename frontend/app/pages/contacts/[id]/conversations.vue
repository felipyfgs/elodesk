<script setup lang="ts">
import type { Contact } from '~/stores/contacts'
import { useConversationsStore } from '~/stores/conversations'
import { format } from 'date-fns'

defineProps<{
  contact: Contact
  contactId: string
}>()

const { t } = useI18n()
const convsStore = useConversationsStore()

function statusColor(status: string): 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral' {
  const map: Record<string, 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'> = { OPEN: 'success', PENDING: 'warning', RESOLVED: 'neutral', SNOOZED: 'info' }
  return map[status] ?? 'neutral'
}
</script>

<template>
  <UPageCard variant="outline" :title="t('contactDetail.conversations')">
    <p v-if="!convsStore.list.length" class="text-sm text-muted">
      {{ t('common.noResults') }}
    </p>

    <div v-else class="space-y-2">
      <div
        v-for="conv in convsStore.list"
        :key="conv.id"
        class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-[var(--ui-border)]"
      >
        <div class="flex items-center gap-3 min-w-0">
          <UBadge :color="statusColor(conv.status)" variant="subtle">
            {{ conv.status }}
          </UBadge>
          <div class="min-w-0">
            <p class="text-sm font-medium truncate">
              {{ conv.inbox?.name ?? '—' }}
            </p>
            <p class="text-xs text-muted">
              {{ format(new Date(conv.lastActivityAt), 'MMM d, HH:mm') }}
            </p>
          </div>
        </div>
        <UButton
          size="xs"
          variant="ghost"
          icon="i-lucide-message-square"
          :to="`/conversations?thread=${conv.id}`"
        />
      </div>
    </div>
  </UPageCard>
</template>
