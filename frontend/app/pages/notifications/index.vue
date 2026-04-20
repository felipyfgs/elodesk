<script setup lang="ts">
import NotificationItem from '~/components/notifications/NotificationItem.vue'
import { useNotificationsStore, type Notification } from '~/stores/notifications'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const store = useNotificationsStore()

const tab = ref<'unread' | 'all'>('unread')
const tabs = computed(() => [
  { label: t('nav.notifications'), value: 'all' },
  { label: t('settings.agents.pending'), value: 'unread' }
])

async function load() {
  await store.fetchRecent(50, tab.value === 'unread')
}

watch(tab, load)
onMounted(load)

async function onItemClick(n: Notification) {
  if (!n.readAt) await store.markRead(n.id)
}

async function markAll() {
  await store.markAllRead()
}
</script>

<template>
  <UDashboardPanel id="notifications">
    <template #header>
      <UDashboardNavbar :title="t('nav.notifications')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UButton
            v-if="store.unreadCount > 0"
            variant="outline"
            icon="i-lucide-check-check"
            @click="markAll"
          >
            {{ t('nav.notifications') }}: {{ store.unreadCount }}
          </UButton>
        </template>
      </UDashboardNavbar>
      <UDashboardToolbar>
        <UTabs v-model="tab" :items="tabs" />
      </UDashboardToolbar>
    </template>
    <template #body>
      <div class="max-w-2xl mx-auto w-full space-y-1">
        <p v-if="!store.loading && store.items.length === 0" class="text-center text-muted py-12">
          {{ t('nav.noNotifications') }}
        </p>
        <NotificationItem
          v-for="n in store.items"
          :key="n.id"
          :notification="n"
          @click="onItemClick"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
