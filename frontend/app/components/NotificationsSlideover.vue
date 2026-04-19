<script setup lang="ts">
import { formatTimeAgo } from '@vueuse/core'
import type { Notification } from '~/types'

const { isNotificationsSlideoverOpen } = useDashboard()
const { t } = useI18n()

// Placeholder — quando o backend expuser `/me/notifications` podemos plugar aqui.
const notifications = ref<Notification[]>([])
</script>

<template>
  <USlideover
    v-model:open="isNotificationsSlideoverOpen"
    :title="t('nav.notifications')"
  >
    <template #body>
      <p v-if="!notifications.length" class="text-sm text-muted p-2">
        {{ t('nav.noNotifications') }}
      </p>
      <NuxtLink
        v-for="n in notifications"
        :key="n.id"
        :to="`/conversations`"
        class="px-3 py-2.5 rounded-md hover:bg-elevated/50 flex items-center gap-3 relative -mx-3 first:-mt-3 last:-mb-3"
      >
        <UChip color="error" :show="!!n.unread" inset>
          <UAvatar v-bind="n.sender.avatar" :alt="n.sender.name" size="md" />
        </UChip>
        <div class="text-sm flex-1">
          <p class="flex items-center justify-between">
            <span class="text-highlighted font-medium">{{ n.sender.name }}</span>
            <time :datetime="n.date" class="text-muted text-xs" v-text="formatTimeAgo(new Date(n.date))" />
          </p>
          <p class="text-dimmed">
            {{ n.body }}
          </p>
        </div>
      </NuxtLink>
    </template>
  </USlideover>
</template>
