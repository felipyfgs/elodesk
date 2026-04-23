<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'
import type { ChannelApiData, ChannelEmailData, ChannelLineData, ChannelTiktokData, ChannelTwilioData, ChannelTwitterData, ChannelWebWidgetData, ChannelWhatsAppData, Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const route = useRoute()
const api = useApi()
const auth = useAuthStore()

const inboxId = computed(() => route.params.id as string)

function formatChannelType(type: string): string {
  const normalized = type.replace('Channel::', '')
  if (normalized === 'FacebookPage') return 'facebook_page'
  if (normalized === 'WebWidget') return 'web_widget'
  return normalized.toLowerCase()
}

const links = computed<NavigationMenuItem[][]>(() => {
  const base = `/accounts/${auth.account?.id}/inboxes/${inboxId.value}`
  const items: NavigationMenuItem[] = [
    { label: t('inboxes.general'), icon: 'i-lucide-layout-dashboard', to: base, exact: true },
    { label: t('inboxes.businessHours'), icon: 'i-lucide-clock', to: `${base}/business-hours` },
    { label: t('inboxes.agents'), icon: 'i-lucide-users', to: `${base}/agents` }
  ]
  return [items]
})

const inbox = ref<Inbox | null>(null)
const loading = ref(true)

async function loadInbox() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const base = await api<Inbox>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}`)
    const channelKey = formatChannelType(base.channelType)
    if (channelKey === 'api') {
      try {
        base.channelApi = await api<ChannelApiData>(`/accounts/${auth.account.id}/inboxes/api/${inboxId.value}`)
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'whatsapp') {
      try {
        const waData = await api<ChannelWhatsAppData>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}/whatsapp`)
        base.channelWhatsApp = waData
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'web_widget') {
      try {
        base.channelWebWidget = await api<ChannelWebWidgetData>(`/accounts/${auth.account.id}/inboxes/web_widget/${inboxId.value}`)
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'email') {
      try {
        base.channelEmail = await api<ChannelEmailData>(`/accounts/${auth.account.id}/inboxes/email/${inboxId.value}`)
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'line') {
      try {
        base.channelLine = await api<ChannelLineData>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}/line`)
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'tiktok') {
      try {
        base.channelTiktok = await api<ChannelTiktokData>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}/tiktok`)
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'twilio') {
      try {
        base.channelTwilio = await api<ChannelTwilioData>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}/twilio`)
      } catch { /* channel data is optional */ }
    } else if (channelKey === 'twitter') {
      try {
        base.channelTwitter = await api<ChannelTwitterData>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}/twitter`)
      } catch { /* channel data is optional */ }
    }
    inbox.value = base
  } catch {
    inbox.value = null
  } finally {
    loading.value = false
  }
}

watch(inboxId, loadInbox, { immediate: true })

const breadcrumb = computed(() => [
  { label: t('inboxes.title'), icon: 'i-lucide-inbox', to: `/accounts/${auth.account?.id}/inboxes` },
  { label: inbox.value?.name ?? `#${inboxId.value}` }
])
</script>

<template>
  <UDashboardPanel id="inbox-detail" :ui="{ body: 'lg:py-12' }">
    <template #header>
      <UDashboardNavbar>
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #title>
          <UBreadcrumb :items="breadcrumb" />
        </template>
        <template #trailing>
          <UBadge
            v-if="inbox"
            :label="t(`inboxes.channels.${formatChannelType(inbox.channelType)}`)"
            variant="subtle"
            size="xs"
          />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <UNavigationMenu :items="links[0]" highlight class="-mx-1 flex-1" />
      </UDashboardToolbar>
    </template>

    <template #body>
      <div class="max-w-6xl mx-auto w-full">
        <div v-if="loading" class="flex flex-col gap-4 py-4">
          <USkeleton class="h-12 w-full rounded-lg" />
          <USkeleton class="h-64 w-full rounded-lg" />
        </div>

        <UEmpty
          v-else-if="!inbox"
          icon="i-lucide-alert-circle"
          :title="t('inboxes.notFound')"
          :ui="{ root: 'py-24' }"
        >
          <template #actions>
            <UButton
              :label="t('inboxes.title')"
              :to="`/accounts/${auth.account?.id}/inboxes`"
            />
          </template>
        </UEmpty>

        <div v-else class="flex flex-col gap-4 sm:gap-6 lg:gap-12 w-full">
          <NuxtPage :inbox="inbox" @inbox-updated="inbox = $event" />
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
