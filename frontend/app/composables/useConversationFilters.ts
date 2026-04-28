import { useStorage } from '@vueuse/core'
import { useConversationsStore, type ConversationSort, type ConversationStatusFilter, type ConversationMeta, type ConversationTab } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

export function useConversationFilters(loadFn: () => Promise<void>) {
  const { t } = useI18n()
  const api = useApi()
  const auth = useAuthStore()
  const convs = useConversationsStore()
  const savedFilters = useSavedFiltersStore()

  const persistedSort = useStorage<ConversationSort>('conversations:sort', 'last_activity_desc')
  const persistedStatus = useStorage<ConversationStatusFilter>('conversations:status', 'OPEN')
  const persistedTab = useStorage<ConversationTab | 'unassigned'>('conversations:tab', 'mine')
  // Legacy migration: 'unassigned' used to be a tab; now it's a separate flag.
  // Carry the intent forward by enabling the flag and switching to 'all'.
  const persistedUnassignedFlag = useStorage<boolean>('conversations:unassignedOnly', false)
  if (persistedTab.value === 'unassigned') {
    persistedTab.value = 'all'
    persistedUnassignedFlag.value = true
  }
  const validTabs: ConversationTab[] = ['mine', 'all']
  if (!validTabs.includes(persistedTab.value as ConversationTab)) persistedTab.value = 'mine'
  const validStatus: ConversationStatusFilter[] = ['OPEN', 'PENDING', 'SNOOZED', 'RESOLVED', 'ALL']
  if (!validStatus.includes(persistedStatus.value)) persistedStatus.value = 'OPEN'

  convs.setFilters({
    sortBy: persistedSort.value,
    status: persistedStatus.value,
    tab: persistedTab.value as ConversationTab,
    unassignedOnly: persistedUnassignedFlag.value
  })

  // Advanced filter state
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
    void loadFn()
  }

  function onAdvancedSaved(filter: SavedFilter) {
    activeSavedFilter.value = filter
    editingFilterId.value = filter.id
  }

  function applySavedFilter(filter: SavedFilter) {
    if (activeSavedFilter.value?.id === filter.id) {
      // Toggle off: clicking the active saved filter again clears it.
      clearAdvancedFilter()
      return
    }
    advancedQuery.value = filter.query
    activeSavedFilter.value = filter
    editingFilterId.value = filter.id
    void loadFn()
  }

  function editActiveFilter() {
    editingFilterId.value = activeSavedFilter.value?.id ?? null
    showAdvancedFilter.value = true
  }

  function clearAdvancedFilter() {
    advancedQuery.value = null
    activeSavedFilter.value = null
    editingFilterId.value = null
    void loadFn()
  }

  function openAdvancedFilter() {
    editingFilterId.value = activeSavedFilter.value?.id ?? null
    showAdvancedFilter.value = true
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
      // best-effort
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

  const displayedList = computed(() => advancedQuery.value ? convs.list : convs.filteredList)

  // Status bucket counters. ALL agrega os 4 buckets — não tem entrada
  // dedicada no `convs.meta`, então somamos por dimensão (mine/unassigned/all)
  // pra alimentar as tabs com o total real.
  const statusBucket = computed(() => {
    const s = convs.filters.status ?? 'OPEN'
    if (s === 'ALL') {
      const m = convs.meta
      return {
        all: m.open.all + m.pending.all + m.resolved.all + m.snoozed.all,
        mine: m.open.mine + m.pending.mine + m.resolved.mine + m.snoozed.mine,
        unassigned: m.open.unassigned + m.pending.unassigned + m.resolved.unassigned + m.snoozed.unassigned
      }
    }
    const map: Record<Exclude<ConversationStatusFilter, 'ALL'>, keyof ConversationMeta> = {
      OPEN: 'open', PENDING: 'pending', RESOLVED: 'resolved', SNOOZED: 'snoozed'
    }
    // Defensive: if `s` is somehow not a known status (stale localStorage,
    // typo in callers, etc.) fall back to OPEN instead of returning undefined.
    return convs.meta[map[s as Exclude<ConversationStatusFilter, 'ALL'>]] ?? convs.meta.open
  })

  function tabBadge(count: number) {
    return { label: String(count), color: 'neutral' as const, variant: 'subtle' as const, size: 'sm' as const }
  }

  // Unread is derived locally from already-loaded conversations — backend has
  // no `unread_count` filter, so the count reflects only the loaded page.
  const unreadCount = computed(() => convs.list.filter(c => (c.unreadCount ?? 0) > 0).length)

  const tabItems = computed(() => [
    { label: t('conversations.sidebar.mine'), value: 'mine', badge: tabBadge(statusBucket.value.mine) },
    { label: t('conversations.sidebar.unread'), value: 'unread', badge: tabBadge(unreadCount.value) },
    { label: t('conversations.sidebar.all'), value: 'all', badge: tabBadge(statusBucket.value.all) }
  ])


  const statusItems = computed(() => [
    { label: t('conversations.status.open'), value: 'OPEN' as const, icon: 'i-lucide-inbox' },
    { label: t('conversations.status.pending'), value: 'PENDING' as const, icon: 'i-lucide-clock' },
    { label: t('conversations.status.snoozed'), value: 'SNOOZED' as const, icon: 'i-lucide-bell-off' },
    { label: t('conversations.status.resolved'), value: 'RESOLVED' as const, icon: 'i-lucide-check-circle-2' },
    { label: t('conversations.status.all'), value: 'ALL' as const, icon: 'i-lucide-list' }
  ])

  const currentStatus = computed(() => {
    const items = statusItems.value
    return items.find(s => s.value === convs.filters.status) ?? items[0]!
  })

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

  function toggleUnattended() {
    const isOn = convs.filters.conversationType === 'unattended'
    convs.setFilters({ conversationType: isOn ? undefined : 'unattended' })
  }

  function selectSort(value: ConversationSort) {
    persistedSort.value = value
    convs.setFilters({ sortBy: value })
  }

  // Two-group dropdown: Status (radio-style — single value) + flags
  // (checkbox — orthogonal toggles). Lets the user combine "Status: Abertas"
  // with "Não atendidas" without juggling separate controls.
  const statusMenuItems = computed(() => [
    statusItems.value.map(item => ({
      label: item.label,
      icon: item.icon,
      type: 'checkbox' as const,
      checked: convs.filters.status === item.value,
      onSelect: () => selectStatus(item.value)
    })),
    [
      {
        label: t('conversations.sidebar.unassigned'),
        icon: 'i-lucide-user-x',
        type: 'checkbox' as const,
        checked: !!convs.filters.unassignedOnly,
        onSelect: () => toggleUnassigned()
      },
      {
        label: t('nav.unattended'),
        icon: 'i-lucide-clock-alert',
        type: 'checkbox' as const,
        checked: convs.filters.conversationType === 'unattended',
        onSelect: () => toggleUnattended()
      }
    ]
  ])

  const sortMenuItems = computed(() =>
    sortItems.value.map(item => ({
      label: item.label,
      icon: item.icon,
      onSelect: () => selectSort(item.value)
    }))
  )

  // UI tabs are: mine / unread / all. The store still tracks `tab` separately
  // (mine / unassigned / all — backend assignee_type) and the new `unread`
  // flag layered on top. The "Sem agente" view moved to the dropdown flags
  // group because it's orthogonal to read-state.
  type UiTab = 'mine' | 'unread' | 'all'
  const activeTab = computed<UiTab>({
    get: () => {
      if (convs.filters.unread) return 'unread'
      return convs.filters.tab === 'mine' ? 'mine' : 'all'
    },
    set: (v) => {
      // Tab choices only mutate `tab` and `unread`. Flags (unassignedOnly,
      // conversationType, etc.) are independent and survive tab switches.
      if (v === 'unread') {
        convs.setFilters({ unread: true, tab: 'all' })
        persistedTab.value = 'all'
        return
      }
      const tab: ConversationTab = v
      persistedTab.value = tab
      convs.setFilters({ tab, unread: false })
    }
  })

  function toggleUnassigned() {
    const next = !convs.filters.unassignedOnly
    persistedUnassignedFlag.value = next
    convs.setFilters({ unassignedOnly: next })
  }

  return {
    advancedQuery,
    showAdvancedFilter,
    activeSavedFilter,
    editingFilterId,
    advancedInitialQuery,
    advancedInitialName,
    displayedList,
    tabItems,
    activeTab,
    statusMenuItems,
    currentStatus,
    sortMenuItems,
    currentSort,
    onAdvancedApply,
    onAdvancedSaved,
    applySavedFilter,
    editActiveFilter,
    clearAdvancedFilter,
    openAdvancedFilter,
    fetchSavedFilters,
    deleteSavedFilter
  }
}
