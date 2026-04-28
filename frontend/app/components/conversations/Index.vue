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
const toast = useToast()
const route = useRoute()
const router = useRouter()

// Selection state is driven by the URL so /accounts/:aid/conversations/:cid
// is shareable and survives reload. `selected` derives from the route param
// and the cached list (or convs.current for deep-fetched threads); writing it
// triggers a router.push so the URL stays in sync.
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
  set: (v: boolean) => {
    if (!v) selectConversation(null)
  }
})

// Deep-link support: when the URL carries a :conversationId that isn't in the
// currently loaded list, fetch it directly. A 404 means the conversation was
// deleted (e.g. another tab) — drop it from the store and route back.
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
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 404) {
      convs.remove(id)
      toast.add({ title: t('conversations.deletedFallback'), icon: 'i-lucide-trash-2', color: 'warning' })
      router.replace(`/accounts/${auth.account.id}/conversations`)
      return
    }
    if (import.meta.dev) console.warn('[conversations] failed to fetch conversation', id, err)
  }
}

watch(selectedId, (id) => {
  ensureSelectedLoaded(id)
})

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('lg')

const filters = useConversationFilters(load)
const { connect: connectRealtime } = useConversationRealtime(selected, () => {
  loadMeta()
})

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
    if (filters.advancedQuery.value) {
      const res = await api<import('~/stores/conversations').ConversationListResponse>(
        `/accounts/${auth.account.id}/conversations/filter`,
        { method: 'POST', body: { query: filters.advancedQuery.value, page: 1, per_page: 100 } }
      )
      if (res.payload) convs.setAll(res.payload)
      if (res.meta) convs.setListMeta(res.meta)
    } else {
      const params: Record<string, string> = { page: '1', per_page: '100', sort_by: convs.filters.sortBy }
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

  const promises: Promise<void>[] = []
  if (!inboxes.list.length) {
    promises.push(api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/inboxes`).then((r) => {
      if (r.payload) inboxes.setAll(r.payload as Inbox[])
    }).catch(() => {}))
  }
  if (!labels.list.length) {
    promises.push(api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/labels`).then((r) => {
      if (r.payload) labels.setAll(r.payload as Label[])
    }).catch(() => {}))
  }
  if (!teams.list.length) {
    promises.push(api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/teams`).then((r) => {
      if (r.payload) teams.setAll(r.payload as Team[])
    }).catch(() => {}))
  }
  if (!agents.items.length) {
    promises.push(agents.fetch().catch(() => {}))
  }
  promises.push(filters.fetchSavedFilters())
  if (!canned.list.length) {
    promises.push(api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/canned_responses`).then((r) => {
      if (r.payload) canned.setAll(r.payload as CannedResponse[])
    }).catch(() => {}))
  }
  await Promise.allSettled(promises)
}

onMounted(async () => {
  await load()
  await ensureSelectedLoaded(selectedId.value)
  connectRealtime()
})

watch(() => convs.filters, () => {
  if (!filters.advancedQuery.value) load()
}, { deep: true })

const filtersBundle = computed(() => ({
  advancedQuery: filters.advancedQuery.value,
  activeSavedFilter: filters.activeSavedFilter.value,
  editingFilterId: filters.editingFilterId.value,
  advancedInitialQuery: filters.advancedInitialQuery.value,
  advancedInitialName: filters.advancedInitialName.value,
  tabItems: filters.tabItems.value,
  statusMenuItems: filters.statusMenuItems.value,
  currentStatus: filters.currentStatus.value,
  sortMenuItems: filters.sortMenuItems.value,
  currentSort: filters.currentSort.value,
  filterMenuItems: filters.filterMenuItems.value,
  hasScopeFilter: filters.hasScopeFilter.value,
  activeFilterSummary: filters.activeFilterSummary.value
}))

function clearScopeFilter() {
  navigateTo(`/accounts/${auth.account?.id}/conversations`)
}
</script>

<template>
  <ConversationsListPanel
    v-model:selected="selected"
    v-model:show-advanced-filter="filters.showAdvancedFilter.value"
    v-model:active-tab="filters.activeTab.value"
    :filters="filtersBundle"
    :displayed-list="filters.displayedList.value"
    :loading="convs.loading"
    @open-advanced-filter="filters.openAdvancedFilter"
    @clear-scope-filter="clearScopeFilter"
    @clear-advanced-filter="filters.clearAdvancedFilter"
    @edit-active-filter="filters.editActiveFilter"
    @delete-saved-filter="(id) => filters.deleteSavedFilter(id)"
    @advanced-apply="filters.onAdvancedApply"
    @advanced-saved="filters.onAdvancedSaved"
  />

  <ConversationsThread v-if="selected && !isMobile" :conversation="selected" @close="selected = null" />
  <div v-else-if="!isMobile" class="hidden lg:flex flex-1 items-center justify-center flex-col gap-2">
    <UIcon name="i-lucide-inbox" class="size-32 text-dimmed" />
    <p class="text-muted">
      {{ t('conversations.select') }}
    </p>
  </div>

  <ClientOnly>
    <USlideover v-if="isMobile" v-model:open="isPanelOpen">
      <template #content>
        <ConversationsThread
          v-if="selected"
          :conversation="selected"
          show-back
          @close="selected = null"
        />
      </template>
    </USlideover>
  </ClientOnly>
</template>
