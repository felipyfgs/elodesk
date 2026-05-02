import { useStorage } from '@vueuse/core'
import { useConversationsStore, STATUS_MAP, type ConversationSort, type ConversationStatusFilter, type ConversationMeta, type ConversationTab } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

const ALL_STATUSES = 'ALL' as const
type StatusUiValue = ConversationStatusFilter | typeof ALL_STATUSES

export function useConversationFilters(loadFn: () => Promise<void>) {
  const { t } = useI18n()
  const api = useApi()
  const auth = useAuthStore()
  const convs = useConversationsStore()
  const savedFilters = useSavedFiltersStore()

  const persistedSort = useStorage<ConversationSort>('conversations:sort', 'last_activity_desc')
  const persistedStatus = useStorage<StatusUiValue>('conversations:status', 'OPEN')
  const persistedTab = useStorage<ConversationTab | 'unassigned'>('conversations:tab', 'mine')
  const persistedUnassignedFlag = useStorage<boolean>('conversations:unassignedOnly', false)
  if (persistedTab.value === 'unassigned') {
    persistedTab.value = 'all'
    persistedUnassignedFlag.value = true
  }
  const persistedUnread = useStorage<boolean>('conversations:unread', false)
  const persistedUnattended = useStorage<boolean>('conversations:unattended', false)
  const validTabs: ConversationTab[] = ['mine', 'all']
  if (!validTabs.includes(persistedTab.value as ConversationTab)) persistedTab.value = 'mine'
  const validStatus: StatusUiValue[] = ['OPEN', 'PENDING', 'SNOOZED', 'RESOLVED', ALL_STATUSES]
  if (!validStatus.includes(persistedStatus.value)) persistedStatus.value = 'OPEN'

  convs.setFilters({
    sortBy: persistedSort.value,
    status: persistedStatus.value === ALL_STATUSES ? undefined : persistedStatus.value,
    tab: persistedTab.value as ConversationTab,
    unassignedOnly: persistedUnassignedFlag.value,
    unread: persistedUnread.value,
    conversationType: persistedUnattended.value ? 'unattended' : undefined
  })

  const persistedManuallyUnread = useStorage<string[]>('conversations:manuallyUnread', [])
  convs.setManuallyUnread(persistedManuallyUnread.value)
  watch(() => convs.manuallyUnread, (v) => {
    persistedManuallyUnread.value = [...v]
  }, { deep: true })

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
      clearAdvancedFilter()
      return
    }
    advancedQuery.value = filter.query ? JSON.parse(filter.query) as FilterQueryPayload : null
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
    } catch { void 0 }
  }

  async function deleteSavedFilter(id: string) {
    if (!auth.account?.id) return
    try {
      await api(`/accounts/${auth.account.id}/custom_filters/${id}`, { method: 'DELETE' })
      savedFilters.remove(id)
      if (activeSavedFilter.value?.id === id) clearAdvancedFilter()
    } catch { void 0 }
  }

  const displayedList = computed(() => advancedQuery.value ? convs.list : convs.filteredList)

  const STATUS_BUCKET_KEY: Record<ConversationStatusFilter, keyof ConversationMeta> = {
    OPEN: 'open', PENDING: 'pending', RESOLVED: 'resolved', SNOOZED: 'snoozed'
  }
  const statusBucket = computed(() => {
    const s = convs.filters.status
    if (!s) {
      const m = convs.meta
      return {
        all: m.open.all + m.pending.all + m.resolved.all + m.snoozed.all,
        mine: m.open.mine + m.pending.mine + m.resolved.mine + m.snoozed.mine,
        unassigned: m.open.unassigned + m.pending.unassigned + m.resolved.unassigned + m.snoozed.unassigned
      }
    }
    return convs.meta[STATUS_BUCKET_KEY[s]]
  })

  function tabBadge(count: number) {
    return { label: String(count), color: 'neutral' as const, variant: 'subtle' as const, size: 'sm' as const }
  }

  const useLocalTabCounts = computed(() =>
    !!convs.filters.unread || convs.filters.conversationType === 'unattended'
  )

  const localTabCounts = computed(() => {
    const myId = auth.user?.id ? String(auth.user.id) : null
    const statusNum = convs.filters.status ? STATUS_MAP[convs.filters.status] : null
    const wantUnread = !!convs.filters.unread
    const wantUnattended = convs.filters.conversationType === 'unattended'
    const sticky = convs.stickyUnreadId
    const manualSet = new Set(convs.manuallyUnread)
    const counts = { mine: 0, unassigned: 0, all: 0 }
    for (const c of convs.list) {
      if (statusNum !== null && c.status !== statusNum) continue
      if (wantUnread && !((c.unreadCount ?? 0) > 0 || c.id === sticky || manualSet.has(c.id))) continue
      if (wantUnattended && !(!c.firstReplyCreatedAt || !!c.waitingSince)) continue
      counts.all++
      if (myId && String(c.assigneeId) === myId) counts.mine++
      if (!c.assigneeId) counts.unassigned++
    }
    return counts
  })

  const tabItems = computed(() => {
    const c = useLocalTabCounts.value
      ? localTabCounts.value
      : { mine: statusBucket.value.mine, unassigned: statusBucket.value.unassigned, all: statusBucket.value.all }
    return [
      { label: t('conversations.sidebar.mine'), value: 'mine' as const, badge: tabBadge(c.mine) },
      { label: t('conversations.sidebar.unassigned'), value: 'unassigned' as const, badge: tabBadge(c.unassigned) },
      { label: t('conversations.sidebar.all'), value: 'all' as const, badge: tabBadge(c.all) }
    ]
  })

  const statusFlagCount = computed(() => {
    let n = 0
    if (convs.filters.unread) n++
    if (convs.filters.conversationType === 'unattended') n++
    return n
  })

  const flagFilterItems = computed(() => [
    {
      label: t('conversations.sidebar.unread'),
      icon: 'i-lucide-mail',
      type: 'checkbox' as const,
      checked: !!convs.filters.unread,
      onSelect: () => toggleUnread()
    },
    {
      label: t('conversations.sidebar.unattended'),
      icon: 'i-lucide-clock-alert',
      type: 'checkbox' as const,
      checked: convs.filters.conversationType === 'unattended',
      onSelect: () => toggleUnattended()
    }
  ])

  const statusItems = computed<{ label: string, value: StatusUiValue, icon: string }[]>(() => [
    { label: t('conversations.status.open'), value: 'OPEN', icon: 'i-lucide-inbox' },
    { label: t('conversations.status.pending'), value: 'PENDING', icon: 'i-lucide-clock' },
    { label: t('conversations.status.snoozed'), value: 'SNOOZED', icon: 'i-lucide-bell-off' },
    { label: t('conversations.status.resolved'), value: 'RESOLVED', icon: 'i-lucide-check-circle-2' },
    { label: t('conversations.status.all'), value: ALL_STATUSES, icon: 'i-lucide-list' }
  ])

  const currentStatus = computed(() => {
    const target: StatusUiValue = convs.filters.status ?? ALL_STATUSES
    return statusItems.value.find(s => s.value === target)!
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

  function selectStatus(value: StatusUiValue) {
    persistedStatus.value = value
    convs.setFilters({ status: value === ALL_STATUSES ? undefined : value })
  }

  function toggleUnattended() {
    const isOn = convs.filters.conversationType === 'unattended'
    persistedUnattended.value = !isOn
    convs.setFilters({ conversationType: isOn ? undefined : 'unattended' })
  }

  function selectSort(value: ConversationSort) {
    persistedSort.value = value
    convs.setFilters({ sortBy: value })
  }

  const statusMenuItems = computed(() =>
    statusItems.value.map(item => ({
      label: item.label,
      icon: item.icon,
      type: 'checkbox' as const,
      checked: (convs.filters.status ?? ALL_STATUSES) === item.value,
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

  type UiTab = 'mine' | 'unassigned' | 'all'
  const activeTab = computed<UiTab>({
    get: () => {
      if (convs.filters.unassignedOnly) return 'unassigned'
      return convs.filters.tab === 'mine' ? 'mine' : 'all'
    },
    set: (v) => {
      if (v === 'unassigned') {
        persistedTab.value = 'all'
        persistedUnassignedFlag.value = true
        convs.setFilters({ tab: 'all', unassignedOnly: true })
        return
      }
      persistedTab.value = v
      persistedUnassignedFlag.value = false
      convs.setFilters({ tab: v, unassignedOnly: false })
    }
  })

  function toggleUnread() {
    const next = !convs.filters.unread
    persistedUnread.value = next
    convs.setFilters({ unread: next })
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
    flagFilterItems,
    statusFlagCount,
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
