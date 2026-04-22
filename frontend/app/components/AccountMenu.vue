<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'

defineProps<{ collapsed?: boolean }>()

const auth = useAuthStore()
const { t } = useI18n()

const selected = computed(() => auth.account
  ? { label: auth.account.name, avatar: { alt: auth.account.name } }
  : { label: t('nav.noAccount'), avatar: { alt: '—' } }
)

const items = computed<DropdownMenuItem[][]>(() => [
  [{
    label: auth.account?.name ?? t('nav.noAccount'),
    type: 'label'
  }],
  [{
    label: t('nav.settingsAccount'),
    icon: 'i-lucide-cog',
    to: auth.account?.id ? `/accounts/${auth.account.id}/settings/account` : '/settings/account'
  }]
])
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'center', collisionPadding: 12 }"
    :ui="{ content: collapsed ? 'w-40' : 'w-(--reka-dropdown-menu-trigger-width)' }"
  >
    <UButton
      v-bind="{
        ...selected,
        label: collapsed ? undefined : selected.label,
        trailingIcon: collapsed ? undefined : 'i-lucide-chevrons-up-down'
      }"
      color="neutral"
      variant="ghost"
      block
      :square="collapsed"
      class="data-[state=open]:bg-elevated"
      :class="[!collapsed && 'py-2']"
      :ui="{ trailingIcon: 'text-dimmed' }"
    />
  </UDropdownMenu>
</template>
