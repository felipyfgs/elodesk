<script setup lang="ts">
import { breakpointsTailwind, useStorage } from '@vueuse/core'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import type { Message } from '~/stores/messages'
import { useMessagesStore } from '~/stores/messages'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useTeamsStore, type Team } from '~/stores/teams'
import { useCannedResponsesStore, type CannedResponse } from '~/stores/cannedResponses'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const rt = useRealtime()
const auth = useAuthStore()
const convs = useConversationsStore()
const messages = useMessagesStore()
const inboxes = useInboxesStore()
const labels = useLabelsStore()
const teams = useTeamsStore()
const canned = useCannedResponsesStore()

const route = useRoute()
const router = useRouter()

// Panel sizes persisted in localStorage
const sidebarSize = useStorage('conversations:sidebar-size', 15)
const listSize = useStorage('conversations:list-size', 30)

// Tab items
const tabItems = computed(() => [
  { label: t('conversations.sidebar.mine'), value: 'mine' },
  { label: t('conversations.sidebar.unassigned'), value: 'unassigned' },
  { label: t('conversations.sidebar.all'), value: 'all' },
  { label: t('conversations.sidebar.mentions'), value: 'mentions' }
])

// Sync tab from URL query
const activeTab = computed({
  get: () => convs.filters.tab,
  set: (v) => {
    convs.setFilters({ tab: v })
    router.replace({ query: { ...route.query, tab: v } })
  }
})

// Initialize tab from URL
onMounted(() => {
  const tab = route.query.tab as string
  if (tab && ['mine', 'unassigned', 'all', 'mentions'].includes(tab)) {
    convs.setFilters({ tab: tab as 'mine' | 'unassigned' | 'all' | 'mentions' })
  }
})

// Selection state
const selected = ref<Conversation | null>(null)

const isPanelOpen = computed({
  get: () => !!selected.value,
  set: (v: boolean) => { if (!v) selected.value = null }
})

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('lg')

// Load data
async function load() {
  if (!auth.account?.id) return

  convs.loading = true
  try {
    // Load conversations with filters
    const params: Record<string, string> = {
      page: '1',
      per_page: '100'
    }

    if (convs.filters.status) params.status = convs.filters.status
    if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId

    // For mine tab, we need the assignee_id filter
    if (convs.filters.tab === 'mine' && auth.user?.id) {
      params.assignee_id = auth.user.id
    }

    const res = await api<{ payload: Conversation[] }>(`/accounts/${auth.account.id}/conversations?${new URLSearchParams(params).toString()}`)
    if (res.payload) {
      convs.setAll(res.payload)
    }
  } finally {
    convs.loading = false
  }

  // Load supporting data in parallel
  const promises: Promise<void>[] = []

  if (!inboxes.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/inboxes`)
        .then((res) => { if (res.payload) inboxes.setAll(res.payload as Inbox[]) })
        .catch(() => {})
    )
  }

  if (!labels.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/labels`)
        .then((res) => { if (res.payload) labels.setAll(res.payload as Label[]) })
        .catch(() => {})
    )
  }

  if (!teams.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/teams`)
        .then((res) => { if (res.payload) teams.setAll(res.payload as Team[]) })
        .catch(() => {})
    )
  }

  if (!canned.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/canned_responses`)
        .then((res) => { if (res.payload) canned.setAll(res.payload as CannedResponse[]) })
        .catch(() => {})
    )
  }

  await Promise.allSettled(promises)
}

onMounted(async () => {
  await load()
  if (auth.account?.id) rt.joinAccount(auth.account.id)

  rt.on<Conversation>('conversation.new', c => convs.upsert(c))
  rt.on<Conversation>('conversation.updated', (c) => {
    convs.upsert(c)
    if (selected.value?.id === c.id) selected.value = c
  })
  rt.on<Message>('message.new', m => messages.upsert(m))
  rt.on<Message>('message.updated', m => messages.upsert(m))
})

watch(selected, (c) => {
  if (c) rt.joinConversation(c.id)
})

watch(() => convs.filteredList, () => {
  if (!convs.filteredList.find(c => c.id === selected.value?.id)) {
    selected.value = null
  }
})

// Re-fetch when filters change
watch(() => convs.filters, () => {
  load()
}, { deep: true })
</script>

<template>
  <!-- Sidebar panel -->
  <UDashboardPanel
    id="conversations-sidebar"
    :default-size="sidebarSize"
    :min-size="10"
    :max-size="25"
    resizable
    collapsible
  >
    <UDashboardNavbar :title="t('conversations.sidebar.title')">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
    </UDashboardNavbar>
    <ConversationsSidebar />
  </UDashboardPanel>

  <!-- List panel -->
  <UDashboardPanel
    id="conversations-list"
    :default-size="listSize"
    :min-size="20"
    :max-size="40"
    resizable
  >
    <UDashboardNavbar :title="t('conversations.title')">
      <template #trailing>
        <UBadge :label="convs.filteredList.length" variant="subtle" />
      </template>
      <template #right>
        <UTabs
          v-model="activeTab"
          :items="tabItems"
          :content="false"
          size="xs"
        />
      </template>
    </UDashboardNavbar>

    <!-- Bulk toolbar -->
    <ConversationsToolbar />

    <!-- Selection header -->
    <div v-if="convs.hasSelection" class="flex items-center justify-between px-4 py-2 bg-primary/5 border-b border-default">
      <label class="flex items-center gap-2 text-sm cursor-pointer">
        <UCheckbox
          :model-value="convs.selection.length === convs.filteredList.length && convs.filteredList.length > 0"
          @update:model-value="(v: boolean | string) => v ? convs.selectAll() : convs.clearSelection()"
        />
        <span class="text-muted">
          {{ t('conversations.bulk.selectAll') }}
        </span>
      </label>
    </div>

    <!-- Conversation list -->
    <div v-if="convs.loading" class="flex items-center justify-center py-12">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>
    <ConversationsList
      v-else-if="convs.filteredList.length"
      v-model="selected"
      :items="convs.filteredList"
    />
    <div v-else class="flex flex-col items-center justify-center py-12 gap-2">
      <UIcon name="i-lucide-message-circle-off" class="size-12 text-dimmed" />
      <p class="text-muted text-sm">
        {{ t('conversations.empty') }}
      </p>
    </div>
  </UDashboardPanel>

  <!-- Thread panel (desktop) -->
  <ConversationThread
    v-if="selected && !isMobile"
    :conversation="selected"
    @close="selected = null"
  />
  <div v-else-if="!isMobile" class="hidden lg:flex flex-1 items-center justify-center flex-col gap-2">
    <UIcon name="i-lucide-inbox" class="size-32 text-dimmed" />
    <p class="text-muted">
      {{ t('conversations.select') }}
    </p>
  </div>

  <!-- Thread panel (mobile slideover) -->
  <ClientOnly>
    <USlideover v-if="isMobile" v-model:open="isPanelOpen">
      <template #content>
        <ConversationThread
          v-if="selected"
          :conversation="selected"
          @close="selected = null"
        />
      </template>
    </USlideover>
  </ClientOnly>
</template>
