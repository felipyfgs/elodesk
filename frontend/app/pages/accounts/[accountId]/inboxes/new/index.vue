<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const runtimeConfig = useRuntimeConfig()
const auth = useAuthStore()

interface ChannelOption {
  kind: string
  icon: string
  color: string
  bg: string
  description: string
}

const twitterEnabled = computed<boolean>(() => {
  const raw = runtimeConfig.public?.featureChannelTwitter
  return raw === true || raw === 'true'
})

const channels = computed<ChannelOption[]>(() => {
  const base: ChannelOption[] = [
    { kind: 'api', icon: 'i-lucide-webhook', color: 'text-blue-500', bg: 'bg-blue-500/10 ring-blue-500/25', description: 'REST API inbox' },
    { kind: 'whatsapp', icon: 'i-simple-icons-whatsapp', color: 'text-green-500', bg: 'bg-green-500/10 ring-green-500/25', description: 'Cloud API or 360dialog' },
    { kind: 'twilio', icon: 'i-lucide-cloud', color: 'text-red-500', bg: 'bg-red-500/10 ring-red-500/25', description: 'Twilio (WhatsApp / SMS)' },
    { kind: 'sms', icon: 'i-lucide-message-square', color: 'text-purple-500', bg: 'bg-purple-500/10 ring-purple-500/25', description: 'SMS via Bandwidth / Zenvia' },
    { kind: 'instagram', icon: 'i-simple-icons-instagram', color: 'text-pink-500', bg: 'bg-pink-500/10 ring-pink-500/25', description: 'Instagram Direct' },
    { kind: 'facebook_page', icon: 'i-simple-icons-facebook', color: 'text-blue-600', bg: 'bg-blue-600/10 ring-blue-600/25', description: 'Facebook Messenger' },
    { kind: 'telegram', icon: 'i-simple-icons-telegram', color: 'text-sky-500', bg: 'bg-sky-500/10 ring-sky-500/25', description: 'Telegram Bot' },
    { kind: 'line', icon: 'i-simple-icons-line', color: 'text-green-600', bg: 'bg-green-600/10 ring-green-600/25', description: 'LINE Messaging API' },
    { kind: 'tiktok', icon: 'i-simple-icons-tiktok', color: 'text-neutral-800', bg: 'bg-neutral-800/10 ring-neutral-800/25', description: 'TikTok Business Messaging' },
    { kind: 'web_widget', icon: 'i-lucide-globe', color: 'text-amber-500', bg: 'bg-amber-500/10 ring-amber-500/25', description: 'Live chat widget' },
    { kind: 'email', icon: 'i-lucide-mail', color: 'text-orange-500', bg: 'bg-orange-500/10 ring-orange-500/25', description: 'IMAP / SMTP' }
  ]
  if (twitterEnabled.value) {
    base.push({ kind: 'twitter', icon: 'i-simple-icons-x', color: 'text-neutral-900', bg: 'bg-neutral-900/10 ring-neutral-900/25', description: 'Twitter / X DMs' })
  }
  return base
})

const breadcrumb = computed(() => [
  { label: t('inboxes.title'), icon: 'i-lucide-inbox', to: `/accounts/${auth.account?.id}/inboxes` },
  { label: t('inboxes.selectChannel') }
])
</script>

<template>
  <UDashboardPanel id="new-inbox">
    <template #header>
      <UDashboardNavbar>
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #title>
          <UBreadcrumb :items="breadcrumb" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="max-w-6xl mx-auto w-full">
        <UPageGrid class="sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <UPageCard
            v-for="ch in channels"
            :key="ch.kind"
            variant="subtle"
            :to="`/accounts/${auth.account?.id}/inboxes/new/${ch.kind}`"
            :ui="{
              container: 'gap-3',
              wrapper: 'items-start',
              leading: `p-3 rounded-full ring ring-inset ${ch.bg}`
            }"
          >
            <template #leading>
              <UIcon :name="ch.icon" :class="['size-6 shrink-0', ch.color]" />
            </template>

            <template #title>
              <span class="font-medium">{{ t(`inboxes.channels.${ch.kind}`) }}</span>
            </template>

            <template #description>
              <span class="text-xs text-muted">{{ ch.description }}</span>
            </template>
          </UPageCard>
        </UPageGrid>
      </div>
    </template>
  </UDashboardPanel>
</template>
