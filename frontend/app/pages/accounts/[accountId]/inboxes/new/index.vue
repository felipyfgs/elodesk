<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const runtimeConfig = useRuntimeConfig()
const auth = useAuthStore()

interface ChannelOption {
  kind: string
  icon: string
  description: string
  beta?: boolean
}

const twitterEnabled = computed<boolean>(() => {
  const raw = runtimeConfig.public?.featureChannelTwitter
  return raw === true || raw === 'true'
})

const tiktokEnabled = computed<boolean>(() => {
  const raw = runtimeConfig.public?.featureChannelTiktok
  return raw === true || raw === 'true'
})

const channels = computed<ChannelOption[]>(() => {
  const base: ChannelOption[] = [
    { kind: 'api', icon: 'i-lucide-webhook', description: 'REST API inbox' },
    { kind: 'whatsapp', icon: 'i-simple-icons-whatsapp', description: 'Cloud API or 360dialog' },
    { kind: 'twilio', icon: 'i-lucide-cloud', description: 'Twilio (WhatsApp / SMS)' },
    { kind: 'sms', icon: 'i-lucide-message-square', description: 'SMS via Bandwidth / Zenvia' },
    { kind: 'instagram', icon: 'i-simple-icons-instagram', description: 'Instagram Direct' },
    { kind: 'facebook_page', icon: 'i-simple-icons-facebook', description: 'Facebook Messenger' },
    { kind: 'telegram', icon: 'i-simple-icons-telegram', description: 'Telegram Bot' },
    { kind: 'line', icon: 'i-simple-icons-line', description: 'LINE Messaging API' },
    { kind: 'web_widget', icon: 'i-lucide-globe', description: 'Live chat widget' },
    { kind: 'email', icon: 'i-lucide-mail', description: 'IMAP / SMTP' }
  ]
  if (tiktokEnabled.value) {
    base.push({ kind: 'tiktok', icon: 'i-simple-icons-tiktok', description: 'TikTok Business Messaging', beta: true })
  }
  if (twitterEnabled.value) {
    base.push({ kind: 'twitter', icon: 'i-simple-icons-x', description: 'Twitter / X DMs', beta: true })
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
              leading: 'p-3 rounded-full bg-elevated ring ring-default'
            }"
          >
            <template #leading>
              <UIcon :name="ch.icon" class="size-6 shrink-0 text-default" />
            </template>

            <template #title>
              <div class="flex items-center gap-2">
                <span class="font-medium">{{ t(`inboxes.channels.${ch.kind}`) }}</span>
                <UBadge
                  v-if="ch.beta"
                  :label="t('common.beta')"
                  variant="subtle"
                  size="xs"
                />
              </div>
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
