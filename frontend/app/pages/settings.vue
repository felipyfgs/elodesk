<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const { t } = useI18n()

const links = computed<NavigationMenuItem[][]>(() => [[
  { label: t('nav.settingsGeneral'), icon: 'i-lucide-user', to: '/settings', exact: true },
  { label: t('nav.settingsMembers'), icon: 'i-lucide-users', to: '/settings/members' },
  { label: t('settings.labels'), icon: 'i-lucide-tag', to: '/settings/labels' },
  { label: t('settings.teams'), icon: 'i-lucide-users-round', to: '/settings/teams' },
  { label: t('settings.canned'), icon: 'i-lucide-message-square', to: '/settings/canned' },
  { label: t('settings.attributes'), icon: 'i-lucide-sliders-horizontal', to: '/settings/attributes' }
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
      <div class="flex flex-col gap-4 sm:gap-6 lg:gap-12 w-full lg:max-w-3xl mx-auto">
        <NuxtPage />
      </div>
    </template>
  </UDashboardPanel>
</template>
