<script setup lang="ts">
import type { TabsItem } from '@nuxt/ui'
import { useNotificationsStore } from '~/stores/notifications'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const store = useNotificationsStore()

const tab = ref<'unread' | 'all'>('unread')

const tabs = computed<TabsItem[]>(() => [
  {
    label: t('notifications.tabUnread'),
    icon: 'i-lucide-mail-warning',
    value: 'unread'
  },
  {
    label: t('notifications.tabAll'),
    icon: 'i-lucide-inbox',
    value: 'all'
  }
])

async function load() {
  await store.fetchRecent(50, tab.value === 'unread')
}

watch(tab, load)
onMounted(load)

async function markAll() {
  await store.markAllRead()
}

const emptyState = computed(() => tab.value === 'unread'
  ? { title: t('notifications.emptyUnreadTitle'), description: t('notifications.emptyUnread') }
  : { title: t('notifications.empty'), description: t('notifications.emptyDescription') }
)
</script>

<template>
  <UDashboardPanel id="notifications">
    <template #header>
      <UDashboardNavbar :title="t('notifications.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            v-if="store.unreadCount > 0"
            variant="outline"
            color="neutral"
            icon="i-lucide-check-check"
            @click="markAll"
          >
            {{ t('notifications.markAllRead') }} ({{ store.unreadCount }})
          </UButton>
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <UTabs
          v-model="tab"
          :items="tabs"
          variant="link"
          color="primary"
          size="sm"
          :ui="{ trigger: 'flex-none' }"
          class="-mb-1.5"
        />
      </UDashboardToolbar>
    </template>

    <template #body>
      <div class="max-w-3xl mx-auto w-full">
        <NotificationsList
          :empty-title="emptyState.title"
          :empty-description="emptyState.description"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
