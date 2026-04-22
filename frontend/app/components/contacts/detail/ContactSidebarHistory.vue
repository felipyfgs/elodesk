<script setup lang="ts">
import { format } from 'date-fns'
import { useConversationsStore } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const convsStore = useConversationsStore()
const auth = useAuthStore()
const aid = computed(() => auth.account?.id ?? '')

function statusColor(status: number): 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral' {
  const map: Record<number, 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'> = {
    0: 'success',
    2: 'warning',
    1: 'neutral',
    3: 'info'
  }
  return map[status] ?? 'neutral'
}

const STATUS_LABELS: Record<number, string> = {
  0: 'OPEN',
  1: 'RESOLVED',
  2: 'PENDING',
  3: 'SNOOZED'
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-medium text-highlighted">
      {{ t('contactDetail.sidebar.history') }}
    </h3>

    <p v-if="!convsStore.list.length" class="text-sm text-muted">
      {{ t('common.noResults') }}
    </p>

    <div v-else class="flex flex-col gap-2">
      <NuxtLink
        v-for="conv in convsStore.list"
        :key="conv.id"
        :to="aid ? `/accounts/${aid}/conversations?thread=${conv.id}` : `/conversations?thread=${conv.id}`"
        class="flex items-center justify-between gap-2 px-3 py-2 rounded-md border border-default hover:bg-elevated transition-colors"
      >
        <div class="flex items-center gap-2 min-w-0">
          <UBadge :color="statusColor(conv.status)" variant="subtle" size="xs">
            {{ STATUS_LABELS[conv.status] ?? conv.status }}
          </UBadge>
          <div class="min-w-0">
            <p class="text-sm font-medium truncate">
              {{ conv.inbox?.name ?? `#${conv.displayId ?? conv.id}` }}
            </p>
            <p class="text-xs text-muted">
              {{ format(new Date(conv.lastActivityAt), 'MMM d, HH:mm') }}
            </p>
          </div>
        </div>
        <UIcon name="i-lucide-chevron-right" class="size-4 text-dimmed shrink-0" />
      </NuxtLink>
    </div>
  </div>
</template>
