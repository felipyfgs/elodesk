<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const { links, isSidebarOpen } = useDashboard()
const auth = useAuthStore()

function flattenNav(menu: NavigationMenuItem[]): Array<{ id: string, label: string, icon?: string, to?: string }> {
  const out: Array<{ id: string, label: string, icon?: string, to?: string }> = []
  for (const item of menu) {
    const label = String(item.label ?? '')
    if (typeof item.to === 'string') out.push({ id: label, label, icon: item.icon as string | undefined, to: item.to })
    if (item.children) out.push(...flattenNav(item.children as NavigationMenuItem[]))
  }
  return out
}

const aid = computed(() => auth.account?.id ?? '')

const groups = computed(() => [
  {
    id: 'navigate',
    label: t('nav.goTo'),
    items: flattenNav(links.value.flat())
  },
  {
    id: 'create',
    label: t('nav.create'),
    items: [
      { id: 'new-inbox', label: t('inboxes.new'), icon: 'i-lucide-store', to: aid.value ? `/accounts/${aid.value}/inboxes/new` : '/inboxes/new' },
      { id: 'new-contact', label: t('contacts.new'), icon: 'i-lucide-user-plus', to: aid.value ? `/accounts/${aid.value}/contacts` : '/contacts' }
    ]
  }
])
</script>

<template>
  <UDashboardGroup unit="rem">
    <UDashboardSidebar
      id="dashboard"
      v-model:open="isSidebarOpen"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <AccountMenu :collapsed="collapsed" />
      </template>

      <template #default="{ collapsed }">
        <UDashboardSearchButton :collapsed="collapsed" class="bg-transparent ring-default" />

        <UNavigationMenu
          :collapsed="collapsed"
          :items="links[0]"
          orientation="vertical"
          tooltip
          popover
        />

        <UNavigationMenu
          :collapsed="collapsed"
          :items="links[1]"
          orientation="vertical"
          tooltip
          class="mt-4"
        />

        <UNavigationMenu
          :collapsed="collapsed"
          :items="links[2]"
          orientation="vertical"
          tooltip
          class="mt-auto"
        />
      </template>

      <template #footer="{ collapsed }">
        <UserMenu :collapsed="collapsed" />
      </template>
    </UDashboardSidebar>

    <UDashboardSearch :groups="groups" />

    <slot />

    <NotificationsSlideover />
  </UDashboardGroup>
</template>
