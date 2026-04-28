<script setup lang="ts">
import { defineAsyncComponent, type Component } from 'vue'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const route = useRoute()
const auth = useAuthStore()

const VALID_KINDS = [
  'api', 'whatsapp', 'sms', 'instagram',
  'facebook_page', 'telegram', 'web_widget', 'email',
  'line', 'tiktok', 'twilio', 'twitter'
] as const

type ChannelKind = (typeof VALID_KINDS)[number]

const kind = computed<ChannelKind | null>(() => {
  const k = route.params.channelKind as string
  return VALID_KINDS.includes(k as ChannelKind) ? (k as ChannelKind) : null
})

const wizardMap: Record<ChannelKind, Component> = {
  api: defineAsyncComponent(() => import('~/components/inboxes/wizards/ApiWizard.vue')),
  whatsapp: defineAsyncComponent(() => import('~/components/inboxes/wizards/WhatsAppWizard.vue')),
  sms: defineAsyncComponent(() => import('~/components/inboxes/wizards/SmsWizard.vue')),
  instagram: defineAsyncComponent(() => import('~/components/inboxes/wizards/InstagramWizard.vue')),
  facebook_page: defineAsyncComponent(() => import('~/components/inboxes/wizards/FacebookPageWizard.vue')),
  telegram: defineAsyncComponent(() => import('~/components/inboxes/wizards/TelegramWizard.vue')),
  web_widget: defineAsyncComponent(() => import('~/components/inboxes/wizards/WebWidgetWizard.vue')),
  email: defineAsyncComponent(() => import('~/components/inboxes/wizards/EmailWizard.vue')),
  line: defineAsyncComponent(() => import('~/components/inboxes/wizards/LineWizard.vue')),
  tiktok: defineAsyncComponent(() => import('~/components/inboxes/wizards/TiktokWizard.vue')),
  twilio: defineAsyncComponent(() => import('~/components/inboxes/wizards/TwilioWizard.vue')),
  twitter: defineAsyncComponent(() => import('~/components/inboxes/wizards/TwitterWizard.vue'))
}

const ActiveWizard = computed(() => (kind.value ? wizardMap[kind.value] : null))

const breadcrumb = computed(() => {
  const accountId = auth.account?.id
  const items: Array<{ label: string, icon?: string, to?: string }> = [
    { label: t('inboxes.title'), icon: 'i-lucide-inbox', to: `/accounts/${accountId}/inboxes` },
    { label: t('inboxes.selectChannel'), to: `/accounts/${accountId}/inboxes/new` }
  ]
  if (kind.value) {
    items.push({ label: t(`inboxes.channels.${kind.value}`) })
  }
  return items
})
</script>

<template>
  <UDashboardPanel id="new-inbox-channel">
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
        <div v-if="!kind" class="flex flex-1 items-center justify-center py-24 text-muted">
          <div class="text-center">
            <UIcon name="i-lucide-alert-circle" class="size-12 mx-auto text-dimmed" />
            <p class="mt-2">
              {{ t('inboxes.wizards.invalidChannel') }}
            </p>
            <UButton
              :label="t('inboxes.selectChannel')"
              class="mt-4"
              :to="`/accounts/${auth.account?.id}/inboxes/new`"
            />
          </div>
        </div>

        <Suspense v-else>
          <component :is="ActiveWizard" />
          <template #fallback>
            <div class="flex items-center justify-center py-24">
              <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
            </div>
          </template>
        </Suspense>
      </div>
    </template>
  </UDashboardPanel>
</template>
