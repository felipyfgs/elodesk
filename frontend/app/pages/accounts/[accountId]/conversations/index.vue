<script setup lang="ts">
import { breakpointsTailwind, useStorage } from '@vueuse/core'
import { useConversationsStore, type Conversation, type ConversationSort, type ConversationStatusFilter, type ConversationMeta } from '~/stores/conversations'
import type { Message } from '~/stores/messages'
import { useMessagesStore } from '~/stores/messages'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useTeamsStore, type Team } from '~/stores/teams'
import { useAgentsStore } from '~/stores/agents'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
import { useCannedResponsesStore, type CannedResponse } from '~/stores/cannedResponses'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

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
const agents = useAgentsStore()
const savedFilters = useSavedFiltersStore()
const canned = useCannedResponsesStore()

const CHANNEL_ICONS: Record<string, string> = {
  api: 'i-lucide-webhook',
  whatsapp: 'i-simple-icons-whatsapp',
  sms: 'i-lucide-message-square',
  instagram: 'i-simple-icons-instagram',
  facebook_page: 'i-simple-icons-facebook',
  telegram: 'i-simple-icons-telegram',
  web_widget: 'i-lucide-globe',
  email: 'i-lucide-mail'
}

// Panel sizes persisted in localStorage
const listSize = useStorage('conversations:list-size', 30)

// Persisted filter preferences (kept in localStorage so users find the list as they left it).
// Tab follows Chatwoot's pattern: stored locally rather than reflected in the URL.
const persistedSort = useStorage<ConversationSort>('conversations:sort', 'last_activity_desc')
const persistedStatus = useStorage<ConversationStatusFilter>('conversations:status', 'OPEN')
const persistedTab = useStorage<'mine' | 'unassigned' | 'all' | 'mentions'>('conversations:tab', 'mine')

// Status → numeric mapping used by the API
const STATUS_CODE: Record<ConversationStatusFilter, string> = {
  OPEN: '0',
  RESOLVED: '1',
  PENDING: '2',
  SNOOZED: '3'
}

// Tab → backend assignee_type mapping
const ASSIGNEE_TYPE: Record<string, string> = {
  mine: 'mine',
  unassigned: 'unassigned',
  all: 'all',
  mentions: 'all' // mentions dimension lives client-side for now
}

// Hydrate filters from persisted preferences before first load
convs.setFilters({ sortBy: persistedSort.value, status: persistedStatus.value, tab: persistedTab.value })

// Advanced filter state (not persisted — transient per session, matches
// Chatwoot's UX where applied filters reset on reload unless saved as a view).
const showAdvancedFilter = ref(false)
const advancedQuery = ref<FilterQueryPayload | null>(null)
const activeSavedFilter = ref<SavedFilter | null>(null)
const editingFilterId = ref<string | null>(null)

const advancedInitialQuery = computed<FilterQueryPayload | null>(() => advancedQuery.value)
const advancedInitialName = computed(() => activeSavedFilter.value?.name ?? '')

function onAdvancedApply(payload: FilterQueryPayload) {
  advancedQuery.value = payload
  activeSavedFilter.value = null
  editingFilterId.value = null
  void load()
}

function onAdvancedSaved(filter: SavedFilter) {
  activeSavedFilter.value = filter
  editingFilterId.value = filter.id
}

function editActiveFilter() {
  editingFilterId.value = activeSavedFilter.value?.id ?? null
  showAdvancedFilter.value = true
}

function clearAdvancedFilter() {
  advancedQuery.value = null
  activeSavedFilter.value = null
  editingFilterId.value = null
  void load()
}

async function fetchSavedFilters() {
  if (!auth.account?.id) return
  try {
    const list = await api<SavedFilter[]>(`/accounts/${auth.account.id}/custom_filters`, {
      params: { filter_type: 'conversation' }
    })
    savedFilters.setAll([
      ...savedFilters.list.filter(f => f.filterType !== 'conversation'),
      ...list
    ])
  } catch {
    // best-effort — sidebar degrades gracefully
  }
}

async function deleteSavedFilter(id: string) {
  if (!auth.account?.id) return
  try {
    await api(`/accounts/${auth.account.id}/custom_filters/${id}`, { method: 'DELETE' })
    savedFilters.remove(id)
    if (activeSavedFilter.value?.id === id) clearAdvancedFilter()
  } catch {
    // silent
  }
}

// Bypass client-side filters (tab/status/inbox/label/team) when an advanced
// filter is active — the server-side query is the authoritative result set.
const displayedList = computed(() => advancedQuery.value ? convs.list : convs.filteredList)

function openAdvancedFilter() {
  editingFilterId.value = activeSavedFilter.value?.id ?? null
  showAdvancedFilter.value = true
}

// Counter lookup keyed by status bucket
const statusBucket = computed(() => {
  const s = convs.filters.status ?? 'OPEN'
  const map: Record<ConversationStatusFilter, keyof ConversationMeta> = {
    OPEN: 'open', PENDING: 'pending', RESOLVED: 'resolved', SNOOZED: 'snoozed'
  }
  return convs.meta[map[s]]
})

// Tab items with live counters rendered via the Nuxt UI `badge` slot
function tabBadge(count: number) {
  return count > 0 ? { label: String(count), color: 'neutral' as const, variant: 'subtle' as const, size: 'sm' as const } : undefined
}

const tabItems = computed(() => [
  {
    label: t('conversations.sidebar.mine'),
    value: 'mine',
    badge: tabBadge(statusBucket.value.mine)
  },
  {
    label: t('conversations.sidebar.unassigned'),
    value: 'unassigned',
    badge: tabBadge(statusBucket.value.unassigned)
  },
  {
    label: t('conversations.sidebar.all'),
    value: 'all',
    badge: tabBadge(statusBucket.value.all)
  },
  {
    label: t('conversations.sidebar.mentions'),
    value: 'mentions'
  }
])

// Status selector items
const statusItems = computed(() => [
  { label: t('conversations.status.open'), value: 'OPEN' as const, icon: 'i-lucide-inbox' },
  { label: t('conversations.status.pending'), value: 'PENDING' as const, icon: 'i-lucide-clock' },
  { label: t('conversations.status.snoozed'), value: 'SNOOZED' as const, icon: 'i-lucide-bell-off' },
  { label: t('conversations.status.resolved'), value: 'RESOLVED' as const, icon: 'i-lucide-check-circle-2' }
])
const currentStatus = computed(() => {
  const items = statusItems.value
  return items.find(s => s.value === convs.filters.status) ?? items[0]!
})

// Sort selector items
const sortItems = computed(() => [
  { label: t('conversations.sort.lastActivityDesc'), value: 'last_activity_desc' as const, icon: 'i-lucide-arrow-down-wide-narrow' },
  { label: t('conversations.sort.lastActivityAsc'), value: 'last_activity_asc' as const, icon: 'i-lucide-arrow-up-wide-narrow' },
  { label: t('conversations.sort.createdDesc'), value: 'created_desc' as const, icon: 'i-lucide-calendar-arrow-down' },
  { label: t('conversations.sort.createdAsc'), value: 'created_asc' as const, icon: 'i-lucide-calendar-arrow-up' }
])
const currentSort = computed(() => {
  const items = sortItems.value
  return items.find(s => s.value === convs.filters.sortBy) ?? items[0]!
})

function selectStatus(value: ConversationStatusFilter) {
  persistedStatus.value = value
  convs.setFilters({ status: value })
}

function selectSort(value: ConversationSort) {
  persistedSort.value = value
  convs.setFilters({ sortBy: value })
}

const statusMenuItems = computed(() =>
  statusItems.value.map(item => ({
    label: item.label,
    icon: item.icon,
    onSelect: () => selectStatus(item.value)
  }))
)

const sortMenuItems = computed(() =>
  sortItems.value.map(item => ({
    label: item.label,
    icon: item.icon,
    onSelect: () => selectSort(item.value)
  }))
)

// Scope filters (inbox, label, team) — URL-driven via `conversations-scope`
// middleware, surfaced as a single Filters dropdown so the list toolbar stays compact.
const currentInboxName = computed(() => {
  const id = convs.filters.inboxId
  return id ? inboxes.list.find(i => i.id === id)?.name ?? null : null
})
const currentLabelName = computed(() => convs.filters.labelId ?? null)
const currentTeamName = computed(() => {
  const id = convs.filters.teamId
  return id ? teams.list.find(tm => tm.id === id)?.name ?? null : null
})

const hasScopeFilter = computed(() =>
  !!(convs.filters.inboxId || convs.filters.labelId || convs.filters.teamId)
)

const activeFilterSummary = computed(() => {
  const parts: string[] = []
  if (currentInboxName.value) parts.push(currentInboxName.value)
  if (currentLabelName.value) parts.push(currentLabelName.value)
  if (currentTeamName.value) parts.push(currentTeamName.value)
  return parts.join(' · ')
})

const filterMenuItems = computed(() => {
  const items: Array<Record<string, unknown>> = [
    {
      label: t('conversations.sidebar.inboxes'),
      icon: 'i-lucide-inbox',
      children: [
        {
          label: t('conversations.sidebar.all'),
          icon: 'i-lucide-list',
          onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`)
        },
        ...inboxes.list.map(ib => ({
          label: ib.name,
          icon: CHANNEL_ICONS[ib.channelType] ?? 'i-lucide-hash',
          onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations/inbox/${ib.id}`)
        }))
      ]
    },
    {
      label: t('conversations.sidebar.labels'),
      icon: 'i-lucide-tag',
      children: [
        {
          label: t('conversations.sidebar.all'),
          icon: 'i-lucide-list',
          onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`)
        },
        ...labels.list.map(l => ({
          label: l.title,
          icon: 'i-lucide-tag',
          onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations/label/${l.title}`)
        }))
      ]
    },
    {
      label: t('conversations.sidebar.teams'),
      icon: 'i-lucide-users',
      children: [
        {
          label: t('conversations.sidebar.all'),
          icon: 'i-lucide-list',
          onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`)
        },
        ...teams.list.map(tm => ({
          label: tm.name,
          icon: 'i-lucide-users-round',
          onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations/team/${tm.id}`)
        }))
      ]
    }
  ]
  if (hasScopeFilter.value) {
    items.push({ type: 'separator' })
    items.push({
      label: t('conversations.sidebar.clearFilters'),
      icon: 'i-lucide-x',
      onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`)
    })
  }
  return items
})

// Tab persists in localStorage (inbox/label/team filters are driven by the
// route and injected by the `conversations-scope` middleware).
const activeTab = computed({
  get: () => convs.filters.tab,
  set: (v) => {
    persistedTab.value = v
    convs.setFilters({ tab: v })
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
async function loadMeta() {
  if (!auth.account?.id) return
  const params: Record<string, string> = {}
  if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId
  const qs = new URLSearchParams(params).toString()
  const url = `/accounts/${auth.account.id}/conversations/meta${qs ? `?${qs}` : ''}`
  try {
    const res = await api<{ payload: ConversationMeta }>(url)
    if (res.payload) convs.setMeta(res.payload)
  } catch (err) {
    // counters are best-effort; surface to console so issues surface in dev
    if (import.meta.dev) console.warn('[conversations] failed to load meta', err)
  }
}

async function load() {
  if (!auth.account?.id) return

  convs.loading = true
  try {
    if (advancedQuery.value) {
      // Advanced filter takes precedence — basic tab/status/inbox filters are
      // ignored so the composed query is the single source of truth.
      const res = await api<{ payload: Conversation[] }>(
        `/accounts/${auth.account.id}/conversations/filter`,
        {
          method: 'POST',
          body: { query: advancedQuery.value, page: 1, per_page: 100 }
        }
      )
      if (res.payload) convs.setAll(res.payload)
    } else {
      const params: Record<string, string> = {
        page: '1',
        per_page: '100',
        sort_by: convs.filters.sortBy
      }

      if (convs.filters.status) params.status = STATUS_CODE[convs.filters.status]
      if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId

      const assigneeType = ASSIGNEE_TYPE[convs.filters.tab]
      if (assigneeType && assigneeType !== 'all') params.assignee_type = assigneeType

      const res = await api<{ payload: Conversation[] }>(`/accounts/${auth.account.id}/conversations?${new URLSearchParams(params).toString()}`)
      if (res.payload) {
        convs.setAll(res.payload)
      }
    }
  } finally {
    convs.loading = false
  }

  loadMeta()

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

  // Agents feed the advanced filter's assignee picker; fetched here so the
  // modal opens with options already available without a separate spinner.
  if (!agents.items.length) {
    promises.push(agents.fetch().catch(() => {}))
  }

  promises.push(fetchSavedFilters())

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

watch(displayedList, () => {
  if (!displayedList.value.find(c => c.id === selected.value?.id)) {
    selected.value = null
  }
})

// Re-fetch when filters change — skipped while an advanced filter is active
// so basic filter mutations don't override the composed query.
watch(() => convs.filters, () => {
  if (!advancedQuery.value) load()
}, { deep: true })
</script>

<template>
  <!-- List panel -->
  <UDashboardPanel
    id="conversations-list"
    :default-size="listSize"
    :min-size="20"
    :max-size="40"
    resizable
  >
    <UDashboardNavbar :title="t('conversations.title')">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
      <template #trailing>
        <UBadge :label="displayedList.length" variant="subtle" />
      </template>
      <template #right>
        <UDropdownMenu :items="statusMenuItems">
          <UButton
            :label="currentStatus.label"
            :icon="currentStatus.icon"
            trailing-icon="i-lucide-chevrons-up-down"
            color="neutral"
            variant="ghost"
            size="sm"
            :disabled="!!advancedQuery"
          />
        </UDropdownMenu>
        <UDropdownMenu :items="filterMenuItems" :content="{ align: 'start' }">
          <UButton
            icon="i-lucide-filter"
            :aria-label="t('conversations.sidebar.title')"
            :color="hasScopeFilter ? 'primary' : 'neutral'"
            :variant="hasScopeFilter ? 'soft' : 'ghost'"
            size="sm"
            :disabled="!!advancedQuery"
          />
        </UDropdownMenu>
        <UButton
          icon="i-lucide-sliders-horizontal"
          :aria-label="t('savedFilters.advancedFilter')"
          :color="advancedQuery ? 'primary' : 'neutral'"
          :variant="advancedQuery ? 'soft' : 'ghost'"
          size="sm"
          @click="openAdvancedFilter"
        />
        <UDropdownMenu :items="sortMenuItems" :content="{ align: 'start' }">
          <UButton
            :icon="currentSort.icon"
            :aria-label="t('conversations.sort.label')"
            color="neutral"
            variant="ghost"
            size="sm"
            :disabled="!!advancedQuery"
          />
        </UDropdownMenu>
      </template>
    </UDashboardNavbar>

    <div
      v-if="hasScopeFilter && !advancedQuery"
      class="flex items-center justify-between gap-2 px-3 py-1.5 bg-primary/5 border-b border-default text-xs"
    >
      <div class="flex items-center gap-1.5 text-muted min-w-0">
        <UIcon name="i-lucide-filter" class="size-3.5 shrink-0" />
        <span class="truncate">{{ activeFilterSummary }}</span>
      </div>
      <UButton
        :label="t('conversations.sidebar.clearFilters')"
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        @click="navigateTo(`/accounts/${auth.account?.id}/conversations`)"
      />
    </div>

    <div
      v-if="advancedQuery"
      class="flex items-center justify-between gap-2 px-3 py-1.5 bg-primary/5 border-b border-default text-xs"
    >
      <div class="flex items-center gap-1.5 text-muted min-w-0">
        <UIcon name="i-lucide-sliders-horizontal" class="size-3.5 shrink-0" />
        <span class="truncate">
          {{ activeSavedFilter?.name ?? t('savedFilters.advancedFilterActive', { count: advancedQuery.conditions.length }) }}
        </span>
      </div>
      <div class="flex items-center gap-1">
        <UButton
          :label="t('savedFilters.edit')"
          icon="i-lucide-pencil"
          color="neutral"
          variant="ghost"
          size="xs"
          @click="editActiveFilter"
        />
        <UButton
          v-if="activeSavedFilter"
          :label="t('savedFilters.delete')"
          icon="i-lucide-trash-2"
          color="error"
          variant="ghost"
          size="xs"
          @click="activeSavedFilter && deleteSavedFilter(activeSavedFilter.id)"
        />
        <UButton
          :label="t('savedFilters.clearFilter')"
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="xs"
          @click="clearAdvancedFilter"
        />
      </div>
    </div>

    <div class="border-b border-default px-2 py-1">
      <UTabs
        v-model="activeTab"
        :items="tabItems"
        :content="false"
        size="xs"
        class="w-full"
      />
    </div>

    <!-- Bulk toolbar -->
    <ConversationsToolbar />

    <!-- Selection header -->
    <div v-if="convs.hasSelection" class="flex items-center justify-between px-4 py-2 bg-primary/5 border-b border-default">
      <label class="flex items-center gap-2 text-sm cursor-pointer">
        <UCheckbox
          :model-value="convs.selection.length === displayedList.length && displayedList.length > 0"
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
      v-else-if="displayedList.length"
      v-model="selected"
      :items="displayedList"
    />
    <div v-else class="flex flex-col items-center justify-center py-12 gap-2">
      <UIcon name="i-lucide-message-circle-off" class="size-12 text-dimmed" />
      <p class="text-muted text-sm">
        {{ t('conversations.empty') }}
      </p>
    </div>

    <FiltersFilterBuilder
      v-model="showAdvancedFilter"
      filter-type="conversation"
      :initial-query="advancedInitialQuery"
      :initial-name="advancedInitialName"
      :editing-id="editingFilterId"
      @apply="onAdvancedApply"
      @save="onAdvancedSaved"
    />
  </UDashboardPanel>

  <!-- Thread panel (desktop) -->
  <ConversationsConversationThread
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
        <ConversationsConversationThread
          v-if="selected"
          :conversation="selected"
          @close="selected = null"
        />
      </template>
    </USlideover>
  </ClientOnly>
</template>
