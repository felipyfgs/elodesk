<script setup lang="ts">
import type { DropdownMenuItem, TabsItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import { useMessagesStore } from '~/stores/messages'
import { useNotificationsStore, type Notification, type NotificationSortOrder } from '~/stores/notifications'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useNotificationsStore()
const convs = useConversationsStore()
const messages = useMessagesStore()
const toast = useToast()

const tab = ref<'unread' | 'all'>('all')

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
  // Swallow rejection: useApi already redirects to /login on 401.
  try {
    await store.fetchRecent(50, tab.value === 'unread')
  } catch {
    // ignore
  }
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

// Selection lives locally — no URL mirroring. Keeps the user on /notifications
// while the right pane swaps in the same <ConversationsThread> mounted by
// /conversations (full parity: header, sidebar, message list, composer).
const selectedNotificationId = ref<number | null>(null)
const loadingConversation = ref(false)
const loadError = ref<'not_found' | 'failed' | null>(null)

const selectedNotification = computed<Notification | null>(() => {
  const id = selectedNotificationId.value
  if (!id) return null
  return store.items.find(n => n.id === id) ?? null
})

// Resolve the conversation id from either the embedded summary or the
// camelCased payload (apiAdapter rewrites every snake_case key, including
// nested ones, so `conversation_id` arrives as `conversationId`).
function notificationConversationId(n: Notification | null): string | null {
  if (!n) return null
  if (n.conversation?.id !== undefined && n.conversation?.id !== null) {
    return String(n.conversation.id)
  }
  const payload = n.payload as Record<string, unknown> | undefined
  const raw = payload?.conversationId ?? payload?.conversation_id
  if (raw === undefined || raw === null) return null
  return String(raw)
}

const selectedConversationId = computed(() => notificationConversationId(selectedNotification.value))

// `currentConversation` shadows what `<Conversations>` puts into convs.current
// — the same path realtime updates flow through, so the thread stays live.
const currentConversation = computed<Conversation | null>(() => {
  const id = selectedConversationId.value
  if (!id) return null
  if (convs.current && String(convs.current.id) === id) return convs.current
  return convs.list.find(c => String(c.id) === id) ?? null
})

async function loadConversation(id: string) {
  if (!auth.account?.id) return
  // Short-circuit: if the backend stripped the conversation summary from
  // this notification, the conversation is already known-deleted. Skip the
  // GET (would 404) and surface the "removed" empty state directly.
  if (selectedNotification.value && !selectedNotification.value.conversation) {
    loadError.value = 'not_found'
    return
  }
  loadingConversation.value = true
  loadError.value = null
  try {
    const cached = convs.list.find(c => String(c.id) === id) ?? (convs.current?.id === id ? convs.current : null)
    if (cached) {
      convs.setCurrent(cached)
    } else {
      const res = await api<Conversation | { payload: Conversation }>(
        `/accounts/${auth.account.id}/conversations/${id}`
      )
      const conv = (res && typeof res === 'object' && 'payload' in res ? res.payload : res) as Conversation
      if (conv) {
        convs.upsert(conv)
        convs.setCurrent(conv)
      }
    }
    await messages.fetchMessages(id, { freshMs: 30_000 })
  } catch (err) {
    const status = (err as { response?: { status?: number } })?.response?.status
    loadError.value = status === 404 ? 'not_found' : 'failed'
  } finally {
    loadingConversation.value = false
  }
}

watch(selectedConversationId, (id, prev) => {
  if (id === prev) return
  loadError.value = null
  if (id) {
    loadConversation(id)
  } else {
    convs.setCurrent(null)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  // Don't carry the selected conversation back to /conversations on the
  // next route — the user expects a clean state there.
  convs.setCurrent(null)
})

async function selectNotification(n: Notification) {
  if (!n.readAt) {
    store.markRead(n.id).catch(() => {})
  }
  if (!notificationConversationId(n)) {
    toast.add({
      title: t('notifications.noConversation'),
      description: t('notifications.noConversationDescription'),
      icon: 'i-lucide-message-square-off',
      color: 'warning'
    })
    return
  }
  selectedNotificationId.value = n.id
}

function toggleRead(n: Notification) {
  if (n.readAt) store.markUnread(n.id).catch(() => {})
  else store.markRead(n.id).catch(() => {})
}

// Sort dropdown — store.sortOrder is the source of truth; switching refetches
// because the cursor pagination is order-aware on the backend.
const sortItems = computed<DropdownMenuItem[]>(() => [
  {
    label: t('notifications.sortNewest'),
    icon: 'i-lucide-arrow-down-narrow-wide',
    type: 'checkbox' as const,
    checked: store.sortOrder === 'desc',
    onSelect: () => changeSort('desc')
  },
  {
    label: t('notifications.sortOldest'),
    icon: 'i-lucide-arrow-up-narrow-wide',
    type: 'checkbox' as const,
    checked: store.sortOrder === 'asc',
    onSelect: () => changeSort('asc')
  }
])

const sortIcon = computed(() => store.sortOrder === 'asc'
  ? 'i-lucide-arrow-up-narrow-wide'
  : 'i-lucide-arrow-down-narrow-wide')

async function changeSort(order: NotificationSortOrder) {
  if (store.sortOrder === order) return
  store.setSortOrder(order)
  await load()
}

// Keyboard navigation matches conversations/List: arrow keys to step, u to
// toggle read, Esc to close the thread. No j/k — the inbox uses arrow keys.
function moveSelection(delta: number) {
  const items = store.items
  if (!items.length) return
  const currentIdx = items.findIndex(n => n.id === selectedNotificationId.value)
  let next: Notification | undefined
  if (currentIdx === -1) {
    next = delta > 0 ? items[0] : items[items.length - 1]
  } else {
    const target = currentIdx + delta
    if (target < 0 || target >= items.length) return
    next = items[target]
  }
  if (next) selectNotification(next)
}

defineShortcuts({
  arrowdown: () => moveSelection(1),
  arrowup: () => moveSelection(-1),
  u: () => {
    if (selectedNotification.value) toggleRead(selectedNotification.value)
  },
  escape: () => {
    selectedNotificationId.value = null
  }
})
</script>

<template>
  <!--
    Tudo no default slot do UDashboardPanel (navbar, status bar, lista) —
    mesmo padrão de ConversationsListPanel. Misturar #header com default slot
    força um wrapper interno com padding que esmaga a lista.
  -->
  <UDashboardPanel
    id="notifications-list"
    :default-size="28"
    :min-size="22"
    :max-size="40"
    resizable
  >
    <UDashboardNavbar :title="t('notifications.title')">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
      <template #trailing>
        <UBadge :label="store.items.length" variant="subtle" />
      </template>
      <template #right>
        <UTooltip v-if="store.unreadCount > 0" :text="t('notifications.markAllRead')">
          <UButton
            icon="i-lucide-check-check"
            :aria-label="t('notifications.markAllRead')"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="markAll"
          >
            <UBadge
              :label="String(store.unreadCount)"
              size="sm"
              color="primary"
              variant="subtle"
            />
          </UButton>
        </UTooltip>
      </template>
    </UDashboardNavbar>

    <!--
      Status bar mirrors ConversationsStatusBar layout: tabs (Não lidas /
      Todas) à esquerda, sort à direita. Mesma borda + padding para casar
      visualmente com o painel de conversations.
    -->
    <div class="flex items-center justify-between gap-1 border-b border-default px-2 py-1">
      <UTabs
        v-model="tab"
        :items="tabs"
        :content="false"
        variant="link"
        color="primary"
        size="sm"
        :ui="{ trigger: 'flex-none', list: 'border-b-0' }"
        class="-mb-1.5 flex-1 min-w-0"
      />
      <div class="flex shrink-0 items-center gap-1">
        <UDropdownMenu :items="sortItems" :content="{ align: 'end' }">
          <UTooltip :text="t('notifications.sort')">
            <UButton
              :icon="sortIcon"
              :aria-label="t('notifications.sort')"
              color="neutral"
              variant="ghost"
              size="xs"
            />
          </UTooltip>
        </UDropdownMenu>
      </div>
    </div>

    <NotificationsList
      :empty-title="emptyState.title"
      :empty-description="emptyState.description"
      :selected-id="selectedNotificationId"
      @select="selectNotification"
      @toggle-read="toggleRead"
    />
  </UDashboardPanel>

  <!--
    Right pane: full ConversationsThread — same component the conversations
    page mounts. Forced remount on conversation change with :key so internal
    refs (scroll position, message bucket) reset cleanly between selections.
  -->
  <UDashboardPanel id="notifications-thread" class="min-w-0 flex-1">
    <div v-if="!selectedNotification" class="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center">
      <UIcon name="i-lucide-bell" class="size-32 text-dimmed" />
      <p class="text-base text-default">
        {{ t('notifications.selectPrompt') }}
      </p>
      <p class="text-sm text-muted">
        {{ t('notifications.selectPromptDescription') }}
      </p>
    </div>

    <div v-else-if="loadingConversation && !currentConversation" class="flex flex-col gap-2 p-4">
      <USkeleton class="h-12 w-full rounded-md" />
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-md" />
    </div>

    <UEmpty
      v-else-if="loadError === 'not_found'"
      icon="i-lucide-trash-2"
      :title="t('notifications.deletedConversation')"
      :description="t('notifications.deletedConversationDescription')"
      variant="naked"
      class="py-12"
    />

    <UEmpty
      v-else-if="loadError === 'failed'"
      icon="i-lucide-circle-alert"
      :title="t('notifications.loadFailed')"
      variant="naked"
      class="py-12"
    />

    <ConversationsThread
      v-else-if="currentConversation"
      :key="currentConversation.id"
      :conversation="currentConversation"
      @close="selectedNotificationId = null"
    />
  </UDashboardPanel>
</template>
