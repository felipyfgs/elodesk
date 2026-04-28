import { createSharedComposable } from '@vueuse/core'
import type { NavigationMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useTeamsStore, type Team } from '~/stores/teams'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
export type SidebarItem = NavigationMenuItem & {
  meta?: { color?: string }
}

const _useDashboard = () => {
  const route = useRoute()
  const { t } = useI18n()
  const auth = useAuthStore()
  const api = useApi()
  const labelsStore = useLabelsStore()
  const inboxesStore = useInboxesStore()
  const teamsStore = useTeamsStore()
  const filtersStore = useSavedFiltersStore()

  const isSidebarOpen = ref(false)
  const isNotificationsSlideoverOpen = ref(false)

  const aid = computed(() => auth.account?.id ?? '')

  const accountPath = (path = '') => aid.value ? `/accounts/${aid.value}${path}` : (path || '/')
  const isSectionActive = (section: string) => {
    const path = accountPath(`/${section}`)
    return route.path === path || route.path.startsWith(`${path}/`)
  }
  const closeSidebar = () => { isSidebarOpen.value = false }

  // Pre-warm caches used across the dashboard (sidebar tree, filter pickers,
  // label managers, etc.). Each store is fed once when the account resolves.
  async function fetchSidebarData() {
    if (!aid.value) return
    const base = `/accounts/${aid.value}`
    const [labels, inboxes, teams, filters] = await Promise.all([
      api<Label[]>(`${base}/labels`).catch(() => [] as Label[]),
      api<Inbox[]>(`${base}/inboxes`).catch(() => [] as Inbox[]),
      api<Team[]>(`${base}/teams`).catch(() => [] as Team[]),
      api<SavedFilter[]>(`${base}/custom_filters`, {
        query: { filter_type: 'conversation' }
      }).catch(() => [] as SavedFilter[])
    ])
    labelsStore.setAll(labels)
    inboxesStore.setAll(inboxes)
    teamsStore.setAll(teams)
    filtersStore.setAll(filters)
  }

  watch(aid, (id) => {
    if (id) void fetchSidebarData()
  }, { immediate: true })

  // Scope pickers and the Unattended view live inside the conversations list
  // panel now (ConversationsStatusBar + tabs). The sidebar keeps a flat link.
  const operationLinks = computed<SidebarItem[]>(() => [
    {
      label: t('nav.conversations'),
      icon: 'i-lucide-messages-square',
      to: accountPath('/conversations'),
      active: isSectionActive('conversations'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.contacts'),
      icon: 'i-lucide-contact-round',
      to: accountPath('/contacts'),
      kbds: ['G', 'O'],
      active: isSectionActive('contacts'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.inboxes'),
      icon: 'i-lucide-inbox',
      to: accountPath('/inboxes'),
      kbds: ['G', 'S'],
      active: isSectionActive('inboxes'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.reports'),
      icon: 'i-lucide-chart-no-axes-combined',
      to: accountPath('/reports'),
      active: isSectionActive('reports'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.notifications'),
      icon: 'i-lucide-bell',
      to: accountPath('/notifications'),
      active: isSectionActive('notifications'),
      onSelect: closeSidebar
    }
  ])

  const resourceLinks = computed<SidebarItem[]>(() => [
    {
      label: t('nav.labels'),
      icon: 'i-lucide-tag',
      to: accountPath('/labels'),
      active: isSectionActive('labels'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.macros'),
      icon: 'i-lucide-wand-sparkles',
      to: accountPath('/macros'),
      active: isSectionActive('macros'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.canned'),
      icon: 'i-lucide-message-square-quote',
      to: accountPath('/canned'),
      active: isSectionActive('canned'),
      onSelect: closeSidebar
    },
    {
      label: t('nav.attributes'),
      icon: 'i-lucide-braces',
      to: accountPath('/attributes'),
      active: isSectionActive('attributes'),
      onSelect: closeSidebar
    }
  ])

  const adminLinks = computed<SidebarItem[]>(() => [
    {
      label: t('nav.settings'),
      icon: 'i-lucide-settings',
      to: accountPath('/settings'),
      active: isSectionActive('settings'),
      defaultOpen: true,
      type: 'trigger',
      children: [
        { label: t('nav.settingsAccount'), icon: 'i-lucide-building-2', to: accountPath('/settings/account'), exact: true, onSelect: closeSidebar },
        { label: t('nav.settingsProfile'), icon: 'i-lucide-user', to: accountPath('/settings/profile'), exact: true, onSelect: closeSidebar },
        { label: t('nav.settingsMembers'), icon: 'i-lucide-users', to: accountPath('/settings/members'), onSelect: closeSidebar },
        { label: t('nav.settingsAgents'), icon: 'i-lucide-shield-user', to: accountPath('/settings/agents'), onSelect: closeSidebar },
        { label: t('nav.settingsTeams'), icon: 'i-lucide-users-round', to: accountPath('/settings/teams'), onSelect: closeSidebar },
        { label: t('nav.settingsSla'), icon: 'i-lucide-gauge', to: accountPath('/settings/sla'), onSelect: closeSidebar },
        { label: t('nav.settingsIntegrations'), icon: 'i-lucide-plug-zap', to: accountPath('/settings/integrations'), onSelect: closeSidebar },
        { label: t('nav.settingsAuditLogs'), icon: 'i-lucide-scroll-text', to: accountPath('/settings/audit-logs'), onSelect: closeSidebar }
      ]
    }
  ])

  const links = computed<SidebarItem[][]>(() => [
    operationLinks.value,
    resourceLinks.value,
    adminLinks.value
  ])

  defineShortcuts({
    ...extractShortcuts(links.value, '-'),
    n: () => {
      isNotificationsSlideoverOpen.value = !isNotificationsSlideoverOpen.value
    }
  })

  watch(() => route.fullPath, () => {
    isNotificationsSlideoverOpen.value = false
  })

  return {
    links,
    operationLinks,
    resourceLinks,
    adminLinks,
    isSidebarOpen,
    isNotificationsSlideoverOpen,
    refreshSidebarData: fetchSidebarData
  }
}

export const useDashboard = createSharedComposable(_useDashboard)
