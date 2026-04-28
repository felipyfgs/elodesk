<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const { links } = useDashboard()
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
  <UDashboardSearch :groups="groups" />
</template>
