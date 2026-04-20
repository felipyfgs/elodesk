<script setup lang="ts">
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const inboxes = useInboxesStore()
const rt = useRealtime()

async function loadInboxes() {
  if (!auth.account?.id) return
  inboxes.loading = true
  try {
    const list = await api<Inbox[]>(`/accounts/${auth.account.id}/inboxes`)
    inboxes.setAll(list)
  } finally {
    inboxes.loading = false
  }
}

onMounted(async () => {
  await loadInboxes()
  if (auth.account?.id) rt.joinAccount(auth.account.id)
})
</script>

<template>
  <UDashboardPanel id="inboxes">
    <template #header>
      <UDashboardNavbar :title="t('inboxes.title')" :ui="{ right: 'gap-2' }">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #trailing>
          <UBadge :label="inboxes.list.length" variant="subtle" />
        </template>
        <template #right>
          <UButton
            icon="i-lucide-plus"
            :label="t('inboxes.new')"
            to="/inboxes/new"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="inboxes.loading" class="flex flex-1 items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div v-else-if="!inboxes.list.length" class="flex flex-1 items-center justify-center py-24 text-muted">
        <div class="text-center">
          <UIcon name="i-lucide-inbox" class="size-12 mx-auto text-dimmed" />
          <p class="mt-2">
            {{ t('inboxes.empty') }}
          </p>
          <UButton
            :label="t('inboxes.new')"
            icon="i-lucide-plus"
            class="mt-4"
            to="/inboxes/new"
          />
        </div>
      </div>

      <UPageGrid v-else class="sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <InboxesInboxCard
          v-for="inbox in inboxes.list"
          :key="inbox.id"
          :inbox="inbox"
        />
      </UPageGrid>
    </template>
  </UDashboardPanel>
</template>
