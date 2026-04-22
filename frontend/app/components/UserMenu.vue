<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'

defineProps<{ collapsed?: boolean }>()

const auth = useAuthStore()
const colorMode = useColorMode()
const appConfig = useAppConfig()
const { logout } = useAuth()
const { t } = useI18n()

const colors = ['red', 'orange', 'amber', 'yellow', 'lime', 'green', 'emerald', 'teal', 'cyan', 'sky', 'blue', 'indigo', 'violet', 'purple', 'fuchsia', 'pink', 'rose']
const neutrals = ['slate', 'gray', 'zinc', 'neutral', 'stone']

const user = computed(() => auth.user
  ? { name: auth.user.name, email: auth.user.email, avatar: { alt: auth.user.name } }
  : { name: '—', email: '', avatar: { alt: '—' } }
)

const items = computed<DropdownMenuItem[][]>(() => [
  [{
    type: 'label',
    label: user.value.name,
    avatar: user.value.avatar
  }],
  [{
    label: t('nav.profileSettings'),
    icon: 'i-lucide-user',
    to: auth.account?.id ? `/accounts/${auth.account.id}/settings/profile` : '/settings/profile'
  }, {
    label: t('nav.settingsAccount'),
    icon: 'i-lucide-settings',
    to: auth.account?.id ? `/accounts/${auth.account.id}/settings/account` : '/settings/account'
  }, {
    label: t('nav.keyboardShortcuts'),
    icon: 'i-lucide-keyboard',
    kbds: ['?'],
    onSelect: () => { window.dispatchEvent(new CustomEvent('open-shortcuts')) }
  }],
  [{
    label: t('nav.theme'),
    icon: 'i-lucide-palette',
    children: [{
      label: t('nav.primary'),
      slot: 'chip',
      chip: appConfig.ui.colors.primary,
      content: { align: 'center', collisionPadding: 16 },
      children: colors.map(color => ({
        label: color,
        chip: color,
        slot: 'chip',
        checked: appConfig.ui.colors.primary === color,
        type: 'checkbox',
        onSelect: (e) => {
          e.preventDefault()
          appConfig.ui.colors.primary = color
        }
      }))
    }, {
      label: t('nav.neutral'),
      slot: 'chip',
      chip: appConfig.ui.colors.neutral === 'neutral' ? 'old-neutral' : appConfig.ui.colors.neutral,
      content: { align: 'end', collisionPadding: 16 },
      children: neutrals.map(color => ({
        label: color,
        chip: color === 'neutral' ? 'old-neutral' : color,
        slot: 'chip',
        type: 'checkbox',
        checked: appConfig.ui.colors.neutral === color,
        onSelect: (e) => {
          e.preventDefault()
          appConfig.ui.colors.neutral = color
        }
      }))
    }]
  }, {
    label: t('nav.appearance'),
    icon: 'i-lucide-sun-moon',
    children: [{
      label: t('nav.light'),
      icon: 'i-lucide-sun',
      type: 'checkbox',
      checked: colorMode.value === 'light',
      onSelect(e: Event) {
        e.preventDefault()
        colorMode.preference = 'light'
      }
    }, {
      label: t('nav.dark'),
      icon: 'i-lucide-moon',
      type: 'checkbox',
      checked: colorMode.value === 'dark',
      onSelect(e: Event) {
        e.preventDefault()
        colorMode.preference = 'dark'
      }
    }]
  }],
  [{
    label: t('nav.readDocs'),
    icon: 'i-lucide-book-open',
    to: '/docs',
    target: '_blank'
  }, {
    label: t('nav.changelog'),
    icon: 'i-lucide-scroll-text',
    to: 'https://github.com/elodesk/elodesk/releases',
    target: '_blank'
  }],
  [{
    label: t('auth.logout'),
    icon: 'i-lucide-log-out',
    onSelect: () => { void logout() }
  }]
])
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'center', collisionPadding: 12 }"
    :ui="{ content: collapsed ? 'w-48' : 'w-(--reka-dropdown-menu-trigger-width)' }"
  >
    <UButton
      v-bind="{
        ...user,
        label: collapsed ? undefined : user.name,
        trailingIcon: collapsed ? undefined : 'i-lucide-chevrons-up-down'
      }"
      color="neutral"
      variant="ghost"
      block
      :square="collapsed"
      class="data-[state=open]:bg-elevated"
      :ui="{ trailingIcon: 'text-dimmed' }"
    />

    <template #chip-leading="{ item }">
      <div class="inline-flex items-center justify-center shrink-0 size-5">
        <span
          class="rounded-full ring ring-bg bg-(--chip-light) dark:bg-(--chip-dark) size-2"
          :style="{
            '--chip-light': `var(--color-${(item as { chip: string }).chip}-500)`,
            '--chip-dark': `var(--color-${(item as { chip: string }).chip}-400)`
          }"
        />
      </div>
    </template>
  </UDropdownMenu>
</template>
