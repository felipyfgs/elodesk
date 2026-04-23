<script setup lang="ts">
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'
import ConfirmModal from '~/components/ConfirmModal.vue'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const inboxes = useInboxesStore()
const rt = useRealtime()
const aid = computed(() => auth.account?.id ?? '')
const toast = useToast()
const overlay = useOverlay()

const deleteModal = overlay.create(ConfirmModal)

const channelConfig: Record<string, { icon: string, iconClass: string }> = {
  api: { icon: 'i-lucide-webhook', iconClass: 'text-primary' },
  whatsapp: { icon: 'i-simple-icons-whatsapp', iconClass: 'text-success' },
  sms: { icon: 'i-lucide-message-square', iconClass: 'text-info' },
  instagram: { icon: 'i-simple-icons-instagram', iconClass: 'text-error' },
  facebook_page: { icon: 'i-simple-icons-facebook', iconClass: 'text-primary' },
  telegram: { icon: 'i-simple-icons-telegram', iconClass: 'text-info' },
  web_widget: { icon: 'i-lucide-globe', iconClass: 'text-warning' },
  email: { icon: 'i-lucide-mail', iconClass: 'text-neutral' },
  line: { icon: 'i-lucide-message-circle', iconClass: 'text-success' },
  tiktok: { icon: 'i-simple-icons-tiktok', iconClass: 'text-neutral' },
  twilio: { icon: 'i-lucide-phone', iconClass: 'text-primary' },
  twitter: { icon: 'i-simple-icons-x', iconClass: 'text-info' }
}

const defaultConfig = { icon: 'i-lucide-inbox', iconClass: 'text-primary' }

function formatChannelType(type: string): string {
  if (!type) return ''
  const normalized = type.replace('Channel::', '')
  if (normalized === 'FacebookPage') return 'facebook_page'
  if (normalized === 'WebWidget') return 'web_widget'
  return normalized.toLowerCase()
}

function getChannelConfig(type: string) {
  return channelConfig[formatChannelType(type)] ?? defaultConfig
}

function getChannelLabel(type: string): string {
  const key = formatChannelType(type)
  const label = t(`inboxes.channels.${key}`)
  if (label.startsWith('inboxes.channels.')) {
    return type.replace('Channel::', '')
  }
  return label
}

async function openDeleteModal(inbox: Inbox) {
  const confirmed = await deleteModal.open({
    title: t('inboxes.delete.title'),
    description: t('inboxes.delete.description'),
    itemName: inbox.name,
    confirmValue: inbox.name,
    confirmPlaceholder: t('inboxes.delete.namePlaceholder'),
    confirmLabel: t('common.delete')
  }).result

  if (confirmed && auth.account?.id) {
    try {
      await api(`/accounts/${auth.account.id}/inboxes/${inbox.id}`, { method: 'DELETE' })
      inboxes.remove(inbox.id)
      toast.add({
        title: t('common.success'),
        description: t('inboxes.delete.success', { name: inbox.name }),
        color: 'success'
      })
    } catch {
      toast.add({
        title: t('common.error'),
        description: t('inboxes.delete.error'),
        color: 'error'
      })
    }
  }
}

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
            :to="`/accounts/${auth.account?.id}/inboxes/new`"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="max-w-6xl mx-auto w-full">
        <div v-if="inboxes.loading" class="flex flex-col gap-4 py-4">
          <USkeleton v-for="n in 5" :key="n" class="h-16 w-full rounded-lg" />
        </div>

        <UEmpty
          v-else-if="!inboxes.list.length"
          icon="i-lucide-inbox"
          :title="t('inboxes.empty')"
          :ui="{ root: 'py-24' }"
        >
          <template #actions>
            <UButton
              :label="t('inboxes.new')"
              icon="i-lucide-plus"
              :to="`/accounts/${auth.account?.id}/inboxes/new`"
            />
          </template>
        </UEmpty>

        <div v-else class="divide-y divide-default border-t border-default">
          <div
            v-for="inbox in inboxes.list"
            :key="inbox.id"
            class="flex items-center justify-between gap-4 py-4"
          >
            <div class="min-w-0 flex items-center gap-4">
              <div class="size-10 justify-center bg-elevated rounded-xl ring ring-default border border-default shadow-sm grid place-items-center shrink-0">
                <UIcon
                  :name="getChannelConfig(inbox.channelType).icon"
                  :class="['size-5', getChannelConfig(inbox.channelType).iconClass]"
                />
              </div>

              <div class="min-w-0">
                <NuxtLink
                  :to="`/accounts/${aid}/inboxes/${inbox.id}`"
                  class="block font-medium text-default hover:text-primary truncate"
                >
                  {{ inbox.name }}
                </NuxtLink>
                <span class="text-sm text-muted">
                  {{ getChannelLabel(inbox.channelType) }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-2 shrink-0">
              <UButton
                icon="i-lucide-settings"
                variant="ghost"
                color="neutral"
                size="sm"
                :aria-label="t('inboxes.settings')"
                :to="`/accounts/${aid}/inboxes/${inbox.id}`"
              />
              <UButton
                icon="i-lucide-trash"
                variant="ghost"
                color="error"
                size="sm"
                :aria-label="t('common.delete')"
                @click="openDeleteModal(inbox)"
              />
            </div>
          </div>
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
