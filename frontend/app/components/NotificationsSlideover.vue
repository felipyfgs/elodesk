<script setup lang="ts">
import { useNotificationsStore, type Notification } from '~/stores/notifications'

const { isNotificationsSlideoverOpen } = useDashboard()
const { t } = useI18n()
const realtime = useRealtime()
const store = useNotificationsStore()

realtime.on('notification.new', (payload: Record<string, unknown>) => {
  store.handleRealtime({ type: 'notification.new', payload: payload as unknown as Notification })
})
realtime.on('notification.read', (payload: Record<string, unknown>) => {
  store.handleRealtime({ type: 'notification.read', payload: payload as { id: number } })
})
realtime.on('notification.read_all', () => {
  store.handleRealtime({ type: 'notification.read_all' })
})

async function loadNotifications() {
  if (store.items.length === 0) await store.fetchRecent(25, false)
}

watch(isNotificationsSlideoverOpen, (open) => {
  if (open) loadNotifications()
})

onMounted(() => {
  store.fetchRecent(25, false)
})

async function onItemClick(n: Notification) {
  if (!n.readAt) await store.markRead(n.id)
}

async function markAllRead() {
  await store.markAllRead()
}

defineExpose({ unreadCount: computed(() => store.unreadCount) })
</script>

<template>
  <USlideover
    v-model:open="isNotificationsSlideoverOpen"
    :title="t('nav.notifications')"
  >
    <template #header>
      <div class="flex items-center justify-between w-full">
        <span class="font-semibold">{{ t('nav.notifications') }}</span>
        <UButton
          v-if="store.unreadCount > 0"
          variant="ghost"
          size="sm"
          @click="markAllRead"
        >
          {{ t('nav.notifications') }}
        </UButton>
      </div>
    </template>
    <template #body>
      <p v-if="!store.items.length" class="text-sm text-muted p-2">
        {{ t('nav.noNotifications') }}
      </p>
      <NotificationsItem
        v-for="n in store.items"
        :key="n.id"
        :notification="n"
        @click="onItemClick"
      />
    </template>
  </USlideover>
</template>
