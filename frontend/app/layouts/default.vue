<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const { t } = useI18n()

const open = ref(false)

const links = computed<NavigationMenuItem[][]>(() => [
  [
    { label: t('nav.home'), icon: 'i-lucide-house', to: '/', onSelect: () => { open.value = false } },
    { label: t('nav.sessions'), icon: 'i-lucide-smartphone', to: '/sessions', onSelect: () => { open.value = false } },
    { label: t('nav.conversations'), icon: 'i-lucide-inbox', to: '/conversations', onSelect: () => { open.value = false } },
    { label: t('nav.contacts'), icon: 'i-lucide-users', to: '/contacts', onSelect: () => { open.value = false } },
    {
      label: t('nav.settings'),
      to: '/settings',
      icon: 'i-lucide-settings',
      defaultOpen: true,
      type: 'trigger',
      children: [
        { label: t('nav.settingsGeneral'), to: '/settings', exact: true, onSelect: () => { open.value = false } },
        { label: t('nav.settingsMembers'), to: '/settings/members', onSelect: () => { open.value = false } }
      ]
    }
  ],
  [
    { label: t('nav.docs'), icon: 'i-lucide-book-open', to: 'http://localhost:3001/docs', target: '_blank' }
  ]
])

function flattenNav(menu: NavigationMenuItem[]): Array<{ id: string, label: string, icon?: string, to?: string }> {
  const out: Array<{ id: string, label: string, icon?: string, to?: string }> = []
  for (const item of menu) {
    const label = String(item.label ?? '')
    if (typeof item.to === 'string') out.push({ id: label, label, icon: item.icon as string | undefined, to: item.to })
    if (item.children) out.push(...flattenNav(item.children as NavigationMenuItem[]))
  }
  return out
}

const groups = computed(() => [{
  id: 'links',
  label: t('nav.goTo'),
  items: flattenNav(links.value.flat())
}, {
  id: 'actions',
  label: t('nav.actions'),
  items: [{
    id: 'new-session',
    label: t('sessions.new'),
    icon: 'i-lucide-smartphone',
    to: '/sessions'
  }]
}])
</script>

<template>
  <UDashboardGroup unit="rem">
    <UDashboardSidebar
      id="default"
      v-model:open="open"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <TeamsMenu :collapsed="collapsed" />
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
