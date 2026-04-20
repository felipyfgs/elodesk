<script setup lang="ts">
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useNotificationsStore } from '~/stores/notifications'
import type { Stat } from '~/types'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const { isNotificationsSlideoverOpen } = useDashboard()
const api = useApi()
const auth = useAuthStore()
const inboxes = useInboxesStore()
const convs = useConversationsStore()
const notificationsStore = useNotificationsStore()

onMounted(() => {
  notificationsStore.fetchRecent(10, false)
})

async function load() {
  if (!auth.account?.id) return
  const [inboxList, convRes] = await Promise.all([
    api<Inbox[]>(`/accounts/${auth.account.id}/inboxes`).catch(() => [] as Inbox[]),
    api<{ payload: Conversation[] }>(`/accounts/${auth.account.id}/conversations`).catch(() => ({ payload: [] as Conversation[] }))
  ])
  inboxes.setAll(inboxList)
  convs.setAll(convRes.payload ?? [])
}

onMounted(load)

const stats = computed<Stat[]>(() => {
  const connected = inboxes.list.filter(i => !!i.channelApi).length
  const unread = convs.list.reduce((acc, c) => acc + (c.meta?.unreadCount ?? 0), 0)
  return [
    { title: t('home.stats.sessions'), icon: 'i-lucide-webhook', value: inboxes.list.length, to: '/sessions' },
    { title: t('home.stats.connected'), icon: 'i-lucide-plug', value: connected, to: '/sessions' },
    { title: t('home.stats.conversations'), icon: 'i-lucide-inbox', value: convs.list.length, to: '/conversations' },
    { title: t('home.stats.unread'), icon: 'i-lucide-bell', value: unread, to: '/conversations' }
  ]
})

function contactTitle(c: Conversation) {
  return c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}
</script>

<template>
  <UDashboardPanel id="home">
    <template #header>
      <UDashboardNavbar :title="t('home.title')" :ui="{ right: 'gap-3' }">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UTooltip :text="t('nav.notifications')" :shortcuts="['N']">
            <UButton
              color="neutral"
              variant="ghost"
              square
              @click="isNotificationsSlideoverOpen = true"
            >
              <UChip
                :show="notificationsStore.unreadCount > 0"
                color="error"
                size="sm"
                :text="notificationsStore.unreadCount"
                inset
              >
                <UIcon name="i-lucide-bell" class="size-5 shrink-0" />
              </UChip>
            </UButton>
          </UTooltip>

          <UButton
            icon="i-lucide-plus"
            size="md"
            class="rounded-full"
            to="/sessions"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
        <UPageCard
          v-for="(stat, index) in stats"
          :key="index"
          :icon="stat.icon"
          :title="stat.title"
          :to="stat.to"
          variant="subtle"
          :ui="{
            container: 'gap-y-1.5',
            wrapper: 'items-start',
            leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25 flex-col',
            title: 'font-normal text-muted text-xs uppercase'
          }"
          class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg hover:z-1"
        >
          <span class="text-2xl font-semibold text-highlighted">{{ stat.value }}</span>
        </UPageCard>
      </UPageGrid>

      <UCard class="mt-6" :ui="{ header: 'flex items-center justify-between' }">
        <template #header>
          <h3 class="font-semibold">
            {{ t('home.recent') }}
          </h3>
          <UButton
            variant="ghost"
            size="xs"
            icon="i-lucide-arrow-right"
            trailing
            to="/conversations"
            :label="t('nav.conversations')"
          />
        </template>

        <p v-if="!convs.list.length" class="text-muted text-sm">
          {{ t('home.empty') }}
        </p>
        <ul v-else class="flex flex-col divide-y divide-default">
          <li v-for="c in convs.list.slice(0, 8)" :key="c.id">
            <NuxtLink
              :to="`/conversations/${c.id}`"
              class="flex items-center justify-between py-2 px-2 rounded-md hover:bg-elevated/50 transition-colors"
            >
              <div>
                <p class="font-medium text-sm">
                  {{ contactTitle(c) }}
                </p>
                <p class="text-xs text-muted">
                  {{ c.inbox?.name }}
                </p>
              </div>
              <span class="text-xs text-muted">{{ new Date(c.lastActivityAt).toLocaleString() }}</span>
            </NuxtLink>
          </li>
        </ul>
      </UCard>
    </template>
  </UDashboardPanel>
</template>
