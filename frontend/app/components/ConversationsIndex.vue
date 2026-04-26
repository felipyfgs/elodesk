<script setup lang="ts">
import { breakpointsTailwind } from '@vueuse/core'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useTeamsStore, type Team } from '~/stores/teams'
import { useAgentsStore } from '~/stores/agents'
import { useCannedResponsesStore, type CannedResponse } from '~/stores/cannedResponses'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const convs = useConversationsStore()
const inboxes = useInboxesStore()
const labels = useLabelsStore()
const teams = useTeamsStore()
const agents = useAgentsStore()
const canned = useCannedResponsesStore()

// Selection state is driven by the URL so /accounts/:aid/conversations/:cid
// is shareable and survives reload. `selected` derives from the route param
// and the cached list (or convs.current for deep-fetched threads); writing it
// triggers a router.push so the URL stays in sync. We also mirror it into
// convs.current for cross-page components (e.g. GlobalAudioMiniPlayer).
const route = useRoute()
const router = useRouter()

const selectedId = computed(() => {
  const id = route.params.conversationId
  return typeof id === 'string' && id ? id : null
})

const selected = computed<Conversation | null>({
  get: () => {
    const id = selectedId.value
    if (!id) return null
    if (convs.current && String(convs.current.id) === id) return convs.current
    return convs.list.find(c => String(c.id) === id) ?? null
  },
  set: (c) => { selectConversation(c) }
})

function selectConversation(c: Conversation | null) {
  const accountId = auth.account?.id
  if (!accountId) return
  const target = c
    ? `/accounts/${accountId}/conversations/${c.id}`
    : `/accounts/${accountId}/conversations`
  if (route.fullPath === target) return
  router.push(target)
}

watch(selected, (c) => {
  convs.setCurrent(c)
}, { immediate: true })
onBeforeUnmount(() => {
  convs.setCurrent(null)
})

const isPanelOpen = computed({
  get: () => !!selected.value,
  set: (v: boolean) => { if (!v) selectConversation(null) }
})

// Deep-link support: when the URL carries a :conversationId that isn't in the
// currently loaded list (e.g. status filter excludes it, or the user opened a
// shared link), fetch it directly so the thread can render.
async function ensureSelectedLoaded(id: string | null) {
  if (!id || !auth.account?.id) return
  if (convs.list.find(c => String(c.id) === id)) return
  if (convs.current && String(convs.current.id) === id) return
  try {
    const res = await api<Conversation | { payload: Conversation }>(`/accounts/${auth.account.id}/conversations/${id}`)
    const conv = (res && typeof res === 'object' && 'payload' in res ? res.payload : res) as Conversation
    if (conv) {
      convs.upsert(conv)
      convs.setCurrent(conv)
    }
  } catch (err) {
    if (import.meta.dev) console.warn('[conversations] failed to fetch conversation', id, err)
  }
}

watch(selectedId, (id) => { ensureSelectedLoaded(id) })

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('lg')

// Filters composable
const {
  advancedQuery, showAdvancedFilter, activeSavedFilter, editingFilterId,
  advancedInitialQuery, advancedInitialName, displayedList,
  tabItems, activeTab, statusMenuItems, currentStatus,
  sortMenuItems, currentSort, filterMenuItems,
  hasScopeFilter, activeFilterSummary,
  onAdvancedApply, onAdvancedSaved, editActiveFilter,
  clearAdvancedFilter, openAdvancedFilter,
  fetchSavedFilters, deleteSavedFilter
} = useConversationFilters(load)

// Realtime composable
const { connect: connectRealtime } = useConversationRealtime(selected)

// Data loading
async function loadMeta() {
  if (!auth.account?.id) return
  const params: Record<string, string> = {}
  if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId
  const qs = new URLSearchParams(params).toString()
  const url = `/accounts/${auth.account.id}/conversations/meta${qs ? `?${qs}` : ''}`
  try {
    const res = await api<{ payload: import('~/stores/conversations').ConversationMeta }>(url)
    if (res.payload) convs.setMeta(res.payload)
  } catch (err) {
    if (import.meta.dev) console.warn('[conversations] failed to load meta', err)
  }
}

const STATUS_CODE: Record<string, string> = { OPEN: '0', RESOLVED: '1', PENDING: '2', SNOOZED: '3' }
const ASSIGNEE_TYPE: Record<string, string> = { mine: 'mine', unassigned: 'unassigned', all: 'all' }

async function load() {
  if (!auth.account?.id) return

  convs.loading = true
  try {
    if (advancedQuery.value) {
      const res = await api<import('~/stores/conversations').ConversationListResponse>(
        `/accounts/${auth.account.id}/conversations/filter`,
        { method: 'POST', body: { query: advancedQuery.value, page: 1, per_page: 100 } }
      )
      if (res.payload) convs.setAll(res.payload)
      if (res.meta) convs.setListMeta(res.meta)
    } else {
      const params: Record<string, string> = {
        page: '1',
        per_page: '100',
        sort_by: convs.filters.sortBy
      }
      const statusCode = convs.filters.status ? STATUS_CODE[convs.filters.status] : undefined
      if (statusCode) params.status = statusCode
      if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId
      const assigneeType = ASSIGNEE_TYPE[convs.filters.tab]
      if (assigneeType && assigneeType !== 'all') params.assignee_type = assigneeType

      const res = await api<import('~/stores/conversations').ConversationListResponse>(
        `/accounts/${auth.account.id}/conversations?${new URLSearchParams(params).toString()}`
      )
      if (res.payload) convs.setAll(res.payload)
      if (res.meta) convs.setListMeta(res.meta)
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
  await ensureSelectedLoaded(selectedId.value)
  connectRealtime()
})

watch(() => convs.filters, () => {
  if (!advancedQuery.value) load()
}, { deep: true })
</script>

<template>
  <UDashboardPanel
    id="conversations-list"
    :default-size="22"
    :min-size="22"
    :max-size="22"
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

    <div class="flex items-center gap-1 border-b border-default px-2 py-1">
      <UTabs
        v-model="activeTab"
        :items="tabItems"
        :content="false"
        size="xs"
        class="min-w-0 flex-1"
      />
      <UDropdownMenu :items="filterMenuItems" :content="{ align: 'end' }">
        <UButton
          icon="i-lucide-filter"
          :aria-label="t('conversations.sidebar.title')"
          :color="hasScopeFilter ? 'primary' : 'neutral'"
          :variant="hasScopeFilter ? 'soft' : 'ghost'"
          size="xs"
          :disabled="!!advancedQuery"
        />
      </UDropdownMenu>
      <UButton
        icon="i-lucide-sliders-horizontal"
        :aria-label="t('savedFilters.advancedFilter')"
        :color="advancedQuery ? 'primary' : 'neutral'"
        :variant="advancedQuery ? 'soft' : 'ghost'"
        size="xs"
        @click="openAdvancedFilter"
      />
      <UDropdownMenu :items="sortMenuItems" :content="{ align: 'end' }">
        <UButton
          :icon="currentSort.icon"
          :aria-label="t('conversations.sort.label')"
          color="neutral"
          variant="ghost"
          size="xs"
          :disabled="!!advancedQuery"
        />
      </UDropdownMenu>
    </div>

    <ConversationsToolbar :total="displayedList.length" />

    <div v-if="convs.loading" class="flex items-center justify-center py-12">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>
    <ConversationsList v-else-if="displayedList.length" v-model="selected" :items="displayedList" />
    <div v-else class="flex flex-col items-center justify-center py-12 gap-2">
      <UIcon name="i-lucide-message-circle-off" class="size-12 text-dimmed" />
      <p class="text-muted text-sm">
        {{ t('conversations.empty') }}
      </p>
    </div>

    <GlobalAudioMiniPlayer />

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

  <ConversationsConversationThread v-if="selected && !isMobile" :conversation="selected" @close="selected = null" />
  <div v-else-if="!isMobile" class="hidden lg:flex flex-1 items-center justify-center flex-col gap-2">
    <UIcon name="i-lucide-inbox" class="size-32 text-dimmed" />
    <p class="text-muted">
      {{ t('conversations.select') }}
    </p>
  </div>

  <ClientOnly>
    <USlideover v-if="isMobile" v-model:open="isPanelOpen">
      <template #content>
        <ConversationsConversationThread
          v-if="selected"
          :conversation="selected"
          show-back
          @close="selected = null"
        />
      </template>
    </USlideover>
  </ClientOnly>
</template>
