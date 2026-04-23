import { useStorage } from '@vueuse/core'
import { useConversationsStore, type ConversationSort, type ConversationStatusFilter, type ConversationMeta, type ConversationTab } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore } from '~/stores/inboxes'
import { useLabelsStore } from '~/stores/labels'
import { useTeamsStore } from '~/stores/teams'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

export function useConversationFilters(loadFn: () => Promise<void>) {
  const { t } = useI18n()
  const api = useApi()
  const auth = useAuthStore()
  const convs = useConversationsStore()
  const inboxes = useInboxesStore()
  const labels = useLabelsStore()
  const teams = useTeamsStore()
  const savedFilters = useSavedFiltersStore()

  const persistedSort = useStorage<ConversationSort>('conversations:sort', 'last_activity_desc')
  const persistedStatus = useStorage<ConversationStatusFilter>('conversations:status', 'OPEN')
  const persistedTab = useStorage<ConversationTab>('conversations:tab', 'mine')
  const validTabs: ConversationTab[] = ['mine', 'unassigned', 'all']
  if (!validTabs.includes(persistedTab.value)) persistedTab.value = 'mine'

  convs.setFilters({ sortBy: persistedSort.value, status: persistedStatus.value, tab: persistedTab.value })

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

  // Status bucket counters
  const statusBucket = computed(() => {
    const s = convs.filters.status ?? 'OPEN'
    const map: Record<ConversationStatusFilter, keyof ConversationMeta> = {
      OPEN: 'open', PENDING: 'pending', RESOLVED: 'resolved', SNOOZED: 'snoozed'
    }
    return convs.meta[map[s]]
  })

  function tabBadge(count: number) {
    return count > 0 ? { label: String(count), color: 'neutral' as const, variant: 'subtle' as const, size: 'sm' as const } : undefined
  }

  const tabItems = computed(() => [
    { label: t('conversations.sidebar.mine'), value: 'mine', badge: tabBadge(statusBucket.value.mine) },
    { label: t('conversations.sidebar.unassigned'), value: 'unassigned', badge: tabBadge(statusBucket.value.unassigned) },
    { label: t('conversations.sidebar.all'), value: 'all', badge: tabBadge(statusBucket.value.all) }
  ])

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

  // Scope filters
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

  const filterMenuItems = computed(() => {
    const items: Array<Record<string, unknown>> = [
      {
        label: t('conversations.sidebar.inboxes'),
        icon: 'i-lucide-inbox',
        children: [
          { label: t('conversations.sidebar.all'), icon: 'i-lucide-list', onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`) },
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
          { label: t('conversations.sidebar.all'), icon: 'i-lucide-list', onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`) },
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
          { label: t('conversations.sidebar.all'), icon: 'i-lucide-list', onSelect: () => navigateTo(`/accounts/${auth.account?.id}/conversations`) },
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

  const activeTab = computed({
    get: () => convs.filters.tab,
    set: (v) => {
      persistedTab.value = v
      convs.setFilters({ tab: v })
    }
  })

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
    filterMenuItems,
    hasScopeFilter,
    activeFilterSummary,
    onAdvancedApply,
    onAdvancedSaved,
    editActiveFilter,
    clearAdvancedFilter,
    openAdvancedFilter,
    fetchSavedFilters,
    deleteSavedFilter
  }
}
