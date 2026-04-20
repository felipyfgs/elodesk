<script setup lang="ts">
import { defineAsyncComponent, type Component } from 'vue'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const route = useRoute()

const VALID_KINDS = [
  'api', 'whatsapp', 'sms', 'instagram',
  'facebook_page', 'telegram', 'web_widget', 'email'
] as const

type ChannelKind = (typeof VALID_KINDS)[number]

const kind = computed<ChannelKind | null>(() => {
  const k = route.params.channelKind as string
  return VALID_KINDS.includes(k as ChannelKind) ? (k as ChannelKind) : null
})

const wizardMap: Record<ChannelKind, () => Promise<Component>> = {
  api: () => import('~/components/inboxes/wizards/ApiWizard.vue'),
  whatsapp: () => import('~/components/inboxes/wizards/WhatsAppWizard.vue'),
  sms: () => import('~/components/inboxes/wizards/SmsWizard.vue'),
  instagram: () => import('~/components/inboxes/wizards/InstagramWizard.vue'),
  facebook_page: () => import('~/components/inboxes/wizards/FacebookPageWizard.vue'),
  telegram: () => import('~/components/inboxes/wizards/TelegramWizard.vue'),
  web_widget: () => import('~/components/inboxes/wizards/WebWidgetWizard.vue'),
  email: () => import('~/components/inboxes/wizards/EmailWizard.vue')
}

const ActiveWizard = computed(() => {
  if (!kind.value) return null
  return defineAsyncComponent(wizardMap[kind.value])
})
</script>

<template>
  <UDashboardPanel id="new-inbox-channel">
    <template #header>
      <UDashboardNavbar :title="t('inboxes.selectChannel')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="!kind" class="flex flex-1 items-center justify-center py-24 text-muted">
        <div class="text-center">
          <UIcon name="i-lucide-alert-circle" class="size-12 mx-auto text-dimmed" />
          <p class="mt-2">
            {{ t('inboxes.wizards.invalidChannel') }}
          </p>
          <UButton
            :label="t('inboxes.selectChannel')"
            icon="i-lucide-arrow-left"
            class="mt-4"
            to="/inboxes/new"
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
    </template>
  </UDashboardPanel>
</template>
