<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'
import type { Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const route = useRoute()
const api = useApi()
const auth = useAuthStore()

const inboxId = computed(() => route.params.id as string)

const links = computed<NavigationMenuItem[][]>(() => [
  [
    { label: t('inboxes.general'), icon: 'i-lucide-layout-dashboard', to: `/inboxes/${inboxId.value}`, exact: true },
    { label: t('inboxes.settings'), icon: 'i-lucide-settings', to: `/inboxes/${inboxId.value}/settings` },
    { label: t('inboxes.agents'), icon: 'i-lucide-users', to: `/inboxes/${inboxId.value}/agents` },
    { label: t('inboxes.webhooks'), icon: 'i-lucide-webhook', to: `/inboxes/${inboxId.value}/webhooks` },
    { label: t('inboxes.businessHours'), icon: 'i-lucide-clock', to: `/inboxes/${inboxId.value}/business-hours` }
  ]
])

const inbox = ref<Inbox | null>(null)
const loading = ref(true)

async function loadInbox() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    inbox.value = await api<Inbox>(`/accounts/${auth.account.id}/inboxes/${inboxId.value}`)
  } catch {
    inbox.value = null
  } finally {
    loading.value = false
  }
}

watch(inboxId, loadInbox, { immediate: true })
</script>

<template>
  <UDashboardPanel id="inbox-detail" :ui="{ body: 'lg:py-12' }">
    <template #header>
      <UDashboardNavbar :title="inbox?.name ?? t('inboxes.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #trailing>
          <UBadge
            v-if="inbox"
            :label="t(`inboxes.channels.${inbox.channelType}`)"
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
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div v-else-if="!inbox" class="flex items-center justify-center py-24 text-muted">
        <div class="text-center">
          <UIcon name="i-lucide-alert-circle" class="size-12 mx-auto text-dimmed" />
          <p class="mt-2">
            {{ t('inboxes.notFound') }}
          </p>
          <UButton
            :label="t('inboxes.title')"
            icon="i-lucide-arrow-left"
            class="mt-4"
            to="/inboxes"
          />
        </div>
      </div>

      <div v-else class="flex flex-col gap-4 sm:gap-6 lg:gap-12 w-full lg:max-w-3xl mx-auto">
        <NuxtPage :inbox="inbox" />
      </div>
    </template>
  </UDashboardPanel>
</template>
