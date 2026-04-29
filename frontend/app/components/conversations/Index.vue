<script setup lang="ts">
import { useConversationsStore, STATUS_CODE, type Conversation, type ConversationMeta, type ConversationStatus } from '~/stores/conversations'
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

// O USlideover precisa ficar montado pra evitar race de HMR com o Teleport,
// mas em ≥lg (não-compact) a Thread vira inline e o slideover não pode
// abrir — caso contrário cobre a tela inteira (`sm:max-w-full`) parecendo
// mobile no desktop. Gate via :open ao invés de v-show porque o root do
// USlideover é um componente (DialogRoot) e v-show é silenciosamente
// ignorado.
const compactPanelOpen = computed({
  get: () => isCompact.value && isPanelOpen.value,
  set: (v: boolean) => { isPanelOpen.value = v }
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

// `isCompact` cobre mobile + tablet (qualquer coisa abaixo de lg). Nesse
// modo a Thread vira slideover em cima da lista; em ≥lg as duas colunas
// coexistem lado a lado.
const { isCompact } = useResponsive()

const filters = useConversationFilters(load)
const { connect: connectRealtime } = useConversationRealtime(selected, () => {
  loadMeta()
})

// Backend `/conversations/meta` only knows `inbox_id`. For label / team /
// unattended / advanced-filter scopes we derive counters locally from the
// already-loaded list — otherwise the tab badges would show account-wide
// totals and contradict the filtered list.
const STATUS_TO_BUCKET: Record<ConversationStatus, keyof ConversationMeta> = {
  0: 'open', 1: 'resolved', 2: 'pending', 3: 'snoozed'
}

function isMetaScopeBackendSupported() {
  if (filters.advancedQuery.value) return false
  const f = convs.filters
  if (f.labelIds?.length) return false
  if (f.teamIds?.length) return false
  if (f.conversationType === 'unattended') return false
  return true
}

function computeLocalMeta(): ConversationMeta {
  const meta: ConversationMeta = {
    open: { all: 0, mine: 0, unassigned: 0, unread: 0 },
    pending: { all: 0, mine: 0, unassigned: 0, unread: 0 },
    resolved: { all: 0, mine: 0, unassigned: 0, unread: 0 },
    snoozed: { all: 0, mine: 0, unassigned: 0, unread: 0 }
  }
  let list: Conversation[] = convs.list
  const f = convs.filters
  if (f.inboxIds?.length) {
    const set = new Set(f.inboxIds.map(String))
    list = list.filter(c => set.has(String(c.inboxId)))
  }
  if (f.labelIds?.length) {
    const set = new Set(f.labelIds)
    list = list.filter(c => c.labels?.some(l => set.has(l)))
  }
  if (f.teamIds?.length) {
    const set = new Set(f.teamIds.map(String))
    list = list.filter(c => set.has(String(c.teamId)))
  }
  if (f.conversationType === 'unattended') {
    list = list.filter(c => !c.firstReplyCreatedAt || !!c.waitingSince)
  }
  const myId = auth.user?.id
  for (const c of list) {
    const k = STATUS_TO_BUCKET[c.status]
    if (!k) continue
    const bucket = meta[k]
    bucket.all++
    if (!c.assigneeId) bucket.unassigned++
    else if (myId && String(c.assigneeId) === String(myId)) bucket.mine++
    if ((c.unreadCount ?? 0) > 0) bucket.unread++
  }
  return meta
}

async function loadMeta() {
  if (!auth.account?.id) return

  if (!isMetaScopeBackendSupported()) {
    convs.setMeta(computeLocalMeta())
    return
  }

  const params: Record<string, string> = {}
  // Backend supports a single inbox filter only; meta numbers reflect that
  // when exactly one inbox is selected. With 0 or 2+ selected we leave the
  // count global since the meta endpoint can't aggregate multi-inbox.
  if (convs.filters.inboxIds?.length === 1) params.inbox_id = convs.filters.inboxIds[0]!
  const qs = new URLSearchParams(params).toString()
  const url = `/accounts/${auth.account.id}/conversations/meta${qs ? `?${qs}` : ''}`
  try {
    const res = await api<{ payload: ConversationMeta }>(url)
    if (res.payload) convs.setMeta(res.payload)
  } catch (err) {
    if (import.meta.dev) console.warn('[conversations] failed to load meta', err)
  }
}

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
      // Single inbox: pass to backend so the list comes pre-filtered.
      // Multi or no selection: pull all and let `filteredList` do client-side.
      if (convs.filters.inboxIds?.length === 1) params.inbox_id = convs.filters.inboxIds[0]!
      // Flags ortogonais (unread / unattended) agora são tratadas no backend
      // (whereClause), então a filtragem real acontece no SQL. Mas mantemos o
      // skip de `assignee_type` quando alguma está ligada: o `localTabCounts`
      // calcula Minhas/Sem agente/Todas a partir de `convs.list`, e isso só
      // funciona se a list não estiver pré-particionada por assignee.
      // (Trade-off: mais dados quando flags estão ativas. Conversas ainda são
      // filtradas pelo backend via `unread`/`conversation_type`.)
      const hasOrthogonalFlag = !!convs.filters.unread || convs.filters.conversationType === 'unattended'
      if (!hasOrthogonalFlag) {
        if (convs.filters.unassignedOnly) {
          params.assignee_type = 'unassigned'
        } else {
          const assigneeType = ASSIGNEE_TYPE[convs.filters.tab]
          if (assigneeType && assigneeType !== 'all') params.assignee_type = assigneeType
        }
      }
      if (convs.filters.conversationType) params.conversation_type = convs.filters.conversationType
      if (convs.filters.unread) params.unread = 'true'
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

// `flush: 'post'` defers the reload until after the DOM update tied to the
// route navigation finishes, avoiding `instance.update is not a function`
// when the middleware mutates filters mid-navigation.
watch(() => convs.filters, () => {
  if (!filters.advancedQuery.value) load()
}, { deep: true, flush: 'post' })

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
  flagFilterItems: filters.flagFilterItems.value,
  statusFlagCount: filters.statusFlagCount.value
}))
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
    @clear-advanced-filter="filters.clearAdvancedFilter"
    @edit-active-filter="filters.editActiveFilter"
    @delete-saved-filter="(id) => filters.deleteSavedFilter(id)"
    @advanced-apply="filters.onAdvancedApply"
    @advanced-saved="filters.onAdvancedSaved"
    @apply-saved-filter="filters.applySavedFilter"
  />

  <!--
    Avoid HMR race conditions by keeping both views mounted and toggling
    visibility instead of v-if. When Vite hot-replaces the component tree,
    v-if triggers unmount/recreate that races with async watchers and
    Teleported elements (USlideover), causing:
      "can't access property 'subTree', vnode.component is null"
      "can't access property 'parentNode', node is null"
    v-show preserves the DOM tree across HMR updates and defers actual
    removal to the transition hooks, which are safe.
  -->
  <div v-show="!isCompact" class="flex flex-1">
    <ConversationsThread
      v-if="selected"
      :key="selected.id"
      :conversation="selected"
      @close="selected = null"
    />
    <div
      v-show="!selected"
      class="hidden lg:flex flex-1 items-center justify-center flex-col gap-2"
    >
      <UIcon name="i-lucide-inbox" class="size-32 text-dimmed" />
      <p class="text-muted">
        {{ t('conversations.select') }}
      </p>
    </div>
  </div>

  <USlideover v-model:open="compactPanelOpen" :ui="{ content: 'sm:max-w-full' }">
    <template #content>
      <ConversationsThread
        v-if="selected && isCompact"
        :key="selected.id"
        :conversation="selected"
        show-back
        @close="selected = null"
      />
    </template>
  </USlideover>
</template>
