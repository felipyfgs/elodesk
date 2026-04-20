<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

definePageMeta({ layout: 'dashboard', middleware: ['settings-redirect'] })

const { t } = useI18n()

const links = computed<NavigationMenuItem[][]>(() => [[
  { label: t('nav.settingsProfile'), icon: 'i-lucide-user', to: '/settings/profile' },
  { label: t('nav.settingsAgents'), icon: 'i-lucide-users', to: '/settings/agents' },
  { label: t('nav.settingsMacros'), icon: 'i-lucide-wand', to: '/settings/macros' },
  { label: t('nav.settingsSla'), icon: 'i-lucide-timer', to: '/settings/sla' },
  { label: t('nav.settingsIntegrations'), icon: 'i-lucide-plug', to: '/settings/integrations' },
  { label: t('nav.settingsAuditLogs'), icon: 'i-lucide-file-text', to: '/settings/audit-logs' },
  { label: t('nav.settingsNotifications'), icon: 'i-lucide-bell', to: '/settings/notifications' },
  { label: t('nav.settingsTeams'), icon: 'i-lucide-users-round', to: '/settings/teams' },
  { label: t('nav.settingsLabels'), icon: 'i-lucide-tag', to: '/settings/labels' },
  { label: t('nav.settingsCanned'), icon: 'i-lucide-message-square', to: '/settings/canned' },
  { label: t('nav.settingsAttributes'), icon: 'i-lucide-sliders-horizontal', to: '/settings/attributes' }
]])
</script>

<template>
  <UDashboardPanel id="settings" :ui="{ body: 'lg:py-12' }">
    <template #header>
      <UDashboardNavbar :title="t('settings.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <UNavigationMenu :items="links[0]" highlight class="-mx-1 flex-1" />
      </UDashboardToolbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4 sm:gap-6 lg:gap-12 w-full lg:max-w-4xl mx-auto">
        <NuxtPage />
      </div>
    </template>
  </UDashboardPanel>
</template>
