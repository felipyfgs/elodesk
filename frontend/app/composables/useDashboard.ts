import { createSharedComposable } from '@vueuse/core'
import type { NavigationMenuItem } from '@nuxt/ui'

const _useDashboard = () => {
  const route = useRoute()
  const { t } = useI18n()
  const isSidebarOpen = ref(false)
  const isNotificationsSlideoverOpen = ref(false)

  const links = computed<NavigationMenuItem[][]>(() => [
    [
      {
        label: t('nav.home'),
        icon: 'i-lucide-house',
        to: '/',
        kbds: ['G', 'H'],
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.conversations'),
        icon: 'i-lucide-inbox',
        to: '/conversations',
        kbds: ['G', 'C'],
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.contacts'),
        icon: 'i-lucide-users',
        to: '/contacts',
        kbds: ['G', 'O'],
        onSelect: () => { isSidebarOpen.value = false }
      },
      {
        label: t('nav.inboxes'),
        icon: 'i-lucide-store',
        to: '/inboxes',
        kbds: ['G', 'S'],
        onSelect: () => { isSidebarOpen.value = false }
      }
    ],
    [
      {
        label: t('nav.settings'),
        icon: 'i-lucide-settings',
        to: '/settings',
        defaultOpen: true,
        type: 'trigger',
        children: [
          { label: t('nav.settingsProfile'), to: '/settings/profile', exact: true, onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsAgents'), to: '/settings/agents', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsMacros'), to: '/settings/macros', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsSla'), to: '/settings/sla', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsIntegrations'), to: '/settings/integrations', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsAuditLogs'), to: '/settings/audit-logs', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsNotifications'), to: '/settings/notifications', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsTeams'), to: '/settings/teams', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsLabels'), to: '/settings/labels', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsCanned'), to: '/settings/canned', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsAttributes'), to: '/settings/attributes', onSelect: () => { isSidebarOpen.value = false } },
          { label: t('nav.settingsMembers'), to: '/settings/members', onSelect: () => { isSidebarOpen.value = false } }
        ]
      }
    ],
    [
      {
        label: t('nav.reports'),
        icon: 'i-lucide-chart-bar',
        to: '/reports'
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
