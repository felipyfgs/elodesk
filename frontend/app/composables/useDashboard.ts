import { createSharedComposable } from '@vueuse/core'
import type { NavigationMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'

const _useDashboard = () => {
  const route = useRoute()
  const { t } = useI18n()
  const auth = useAuthStore()
  const isSidebarOpen = ref(false)
  const isNotificationsSlideoverOpen = ref(false)

  const aid = computed(() => auth.account?.id ?? '')

  const accountPath = (path = '') => aid.value ? `/accounts/${aid.value}${path}` : (path || '/')
  const isAccountRootActive = () => route.path === accountPath()
  const isSectionActive = (section: string) => {
    const path = accountPath(`/${section}`)
    return route.path === path || route.path.startsWith(`${path}/`)
  }

  const links = computed<NavigationMenuItem[][]>(() => [
    [
      {
        label: t('nav.home'),
        icon: 'i-lucide-house',
        to: accountPath(),
        kbds: ['G', 'H'],
        active: isAccountRootActive(),
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.conversations'),
        icon: 'i-lucide-messages-square',
        to: accountPath('/conversations'),
        kbds: ['G', 'C'],
        active: isSectionActive('conversations'),
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.contacts'),
        icon: 'i-lucide-contact-round',
        to: accountPath('/contacts'),
        kbds: ['G', 'O'],
        active: isSectionActive('contacts'),
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.inboxes'),
        icon: 'i-lucide-inbox',
        to: accountPath('/inboxes'),
        kbds: ['G', 'S'],
        active: isSectionActive('inboxes'),
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.reports'),
        icon: 'i-lucide-chart-no-axes-combined',
        to: accountPath('/reports'),
        active: isSectionActive('reports'),
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.notifications'),
        icon: 'i-lucide-bell',
        to: accountPath('/notifications'),
        active: isSectionActive('notifications'),
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.settings'),
        icon: 'i-lucide-settings',
        to: accountPath('/settings'),
        active: isSectionActive('settings'),
        defaultOpen: true,
        type: 'trigger',
        children: [
          { label: t('nav.settingsAccount'), icon: 'i-lucide-building-2', to: accountPath('/settings/account'), exact: true, onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsProfile'), icon: 'i-lucide-user', to: accountPath('/settings/profile'), exact: true, onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsMembers'), icon: 'i-lucide-users', to: accountPath('/settings/members'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsAgents'), icon: 'i-lucide-shield-user', to: accountPath('/settings/agents'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsTeams'), icon: 'i-lucide-users-round', to: accountPath('/settings/teams'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsMacros'), icon: 'i-lucide-wand-sparkles', to: accountPath('/settings/macros'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsCanned'), icon: 'i-lucide-message-square-quote', to: accountPath('/settings/canned'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsLabels'), icon: 'i-lucide-tag', to: accountPath('/settings/labels'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsAttributes'), icon: 'i-lucide-braces', to: accountPath('/settings/attributes'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsSla'), icon: 'i-lucide-gauge', to: accountPath('/settings/sla'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsIntegrations'), icon: 'i-lucide-plug-zap', to: accountPath('/settings/integrations'), onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsAuditLogs'), icon: 'i-lucide-scroll-text', to: accountPath('/settings/audit-logs'), onSelect: () => { isSidebarOpen.value = false } }
        ]
      }
    ]
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
    isSidebarOpen,
    isNotificationsSlideoverOpen
  }
}

export const useDashboard = createSharedComposable(_useDashboard)
