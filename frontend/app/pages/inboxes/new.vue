<script setup lang="ts">
definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()

interface ChannelOption {
  kind: string
  icon: string
  color: string
  bg: string
  description: string
}

const channels: ChannelOption[] = [
  { kind: 'api', icon: 'i-lucide-webhook', color: 'text-blue-500', bg: 'bg-blue-500/10 ring-blue-500/25', description: 'REST API inbox' },
  { kind: 'whatsapp', icon: 'i-simple-icons-whatsapp', color: 'text-green-500', bg: 'bg-green-500/10 ring-green-500/25', description: 'WhatsApp Cloud API' },
  { kind: 'sms', icon: 'i-lucide-message-square', color: 'text-purple-500', bg: 'bg-purple-500/10 ring-purple-500/25', description: 'SMS via Twilio / Zenvia' },
  { kind: 'instagram', icon: 'i-simple-icons-instagram', color: 'text-pink-500', bg: 'bg-pink-500/10 ring-pink-500/25', description: 'Instagram Direct' },
  { kind: 'facebook_page', icon: 'i-simple-icons-facebook', color: 'text-blue-600', bg: 'bg-blue-600/10 ring-blue-600/25', description: 'Facebook Messenger' },
  { kind: 'telegram', icon: 'i-simple-icons-telegram', color: 'text-sky-500', bg: 'bg-sky-500/10 ring-sky-500/25', description: 'Telegram Bot' },
  { kind: 'web_widget', icon: 'i-lucide-globe', color: 'text-amber-500', bg: 'bg-amber-500/10 ring-amber-500/25', description: 'Live chat widget' },
  { kind: 'email', icon: 'i-lucide-mail', color: 'text-orange-500', bg: 'bg-orange-500/10 ring-orange-500/25', description: 'IMAP / SMTP' }
]
</script>

<template>
  <UDashboardPanel id="new-inbox">
    <template #header>
      <UDashboardNavbar :title="t('inboxes.selectChannel')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UPageGrid class="sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        <UPageCard
          v-for="ch in channels"
          :key="ch.kind"
          variant="subtle"
          :to="`/inboxes/new/${ch.kind}`"
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
    </template>
  </UDashboardPanel>
</template>
