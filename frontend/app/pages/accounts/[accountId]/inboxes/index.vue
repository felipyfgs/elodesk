<script setup lang="ts">
import { formatTimeAgo } from '@vueuse/core'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const inboxes = useInboxesStore()
const rt = useRealtime()
const aid = computed(() => auth.account?.id ?? '')
const toast = useToast()

const deleteModalOpen = ref(false)
const inboxToDelete = ref<Inbox | null>(null)
const deleteConfirmName = ref('')
const deleting = ref(false)

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

function getBadgeColor(type: string) {
  if (type === 'whatsapp' || type === 'telegram' || type === 'line') return 'success'
  if (type === 'api' || type === 'facebook_page' || type === 'twilio') return 'primary'
  if (type === 'sms' || type === 'web_widget') return 'warning'
  if (type === 'instagram') return 'error'
  if (type === 'email') return 'info'
  return 'neutral'
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

function getOpenConversationsLabel(count?: number): string {
  if (count == null) return '-'
  return t('inboxes.openConversations', { count })
}

function getVisibleAgents(inbox: Inbox) {
  return (inbox.agents ?? []).slice(0, 3)
}

function getExtraAgents(inbox: Inbox): number {
  return Math.max(0, (inbox.agents?.length ?? 0) - 3)
}

const canConfirmDelete = computed(() => {
  return !!inboxToDelete.value && deleteConfirmName.value.trim() === inboxToDelete.value.name
})

function openDeleteModal(inbox: Inbox) {
  inboxToDelete.value = inbox
  deleteConfirmName.value = ''
  deleteModalOpen.value = true
}

function closeDeleteModal() {
  deleteModalOpen.value = false
  inboxToDelete.value = null
  deleteConfirmName.value = ''
}

async function confirmDeleteInbox() {
  if (!auth.account?.id || !inboxToDelete.value || !canConfirmDelete.value) return

  deleting.value = true
  try {
    const target = inboxToDelete.value
    await api(`/accounts/${auth.account.id}/inboxes/${target.id}`, { method: 'DELETE' })
    inboxes.remove(target.id)
    toast.add({
      title: t('common.success'),
      description: t('inboxes.delete.success', { name: target.name }),
      color: 'success'
    })
    closeDeleteModal()
  } catch {
    toast.add({
      title: t('common.error'),
      description: t('inboxes.delete.error'),
      color: 'error'
    })
  } finally {
    deleting.value = false
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
        <div v-if="inboxes.loading" class="flex flex-1 items-center justify-center py-24">
          <UIcon name="i-lucide-loader-circle" class="size-8 animate-spin text-primary" />
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
              :to="`/accounts/${auth.account?.id}/inboxes/new`"
            />
          </div>
        </div>

        <div v-else class="w-full rounded-lg border border-default overflow-hidden">
          <div class="flex items-center justify-between gap-4 px-4 py-2.5 bg-muted text-left">
            <span class="text-xs font-medium text-muted uppercase tracking-wide">
              {{ t('inboxes.fields.name') }}
            </span>
            <span class="text-xs font-medium text-muted uppercase tracking-wide text-right">
              {{ t('inboxes.fields.actions') }}
            </span>
          </div>

          <UPageList class="w-full">
            <div
              v-for="inbox in inboxes.list"
              :key="inbox.id"
              class="flex items-center justify-between gap-4 px-4 py-3.5 border-t border-default"
            >
              <div class="min-w-0 flex items-center gap-3">
                <div class="p-2.5 rounded-lg bg-elevated ring ring-default shrink-0">
                  <UIcon
                    :name="getChannelConfig(inbox.channelType).icon"
                    :class="['size-4', getChannelConfig(inbox.channelType).iconClass]"
                  />
                </div>

                <div class="min-w-0">
                  <NuxtLink
                    :to="`/accounts/${aid}/inboxes/${inbox.id}`"
                    class="block font-medium text-default hover:text-primary truncate"
                  >
                    {{ inbox.name }}
                  </NuxtLink>

                  <div class="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-muted">
                    <UBadge
                      :color="getBadgeColor(formatChannelType(inbox.channelType))"
                      variant="subtle"
                      size="xs"
                    >
                      {{ getChannelLabel(inbox.channelType) }}
                    </UBadge>

                    <span>{{ getOpenConversationsLabel(inbox.openConversationCount) }}</span>

                    <span v-if="inbox.lastActivityAt">•</span>

                    <span v-if="inbox.lastActivityAt">
                      {{ formatTimeAgo(new Date(inbox.lastActivityAt)) }}
                    </span>
                  </div>
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <div
                  v-if="getVisibleAgents(inbox).length"
                  class="flex items-center gap-1"
                >
                  <UAvatarGroup size="xs" :max="3">
                    <UTooltip
                      v-for="agent in getVisibleAgents(inbox)"
                      :key="agent.userId"
                      :text="agent.user?.name"
                    >
                      <UAvatar
                        :src="agent.user?.avatarUrl ?? undefined"
                        :alt="agent.user?.name"
                        size="xs"
                      />
                    </UTooltip>
                  </UAvatarGroup>
                  <span v-if="getExtraAgents(inbox) > 0" class="text-xs text-muted ml-1">
                    +{{ getExtraAgents(inbox) }}
                  </span>
                </div>

                <UButton
                  icon="i-lucide-settings"
                  variant="ghost"
                  color="neutral"
                  size="sm"
                  :aria-label="t('inboxes.settings')"
                  :to="`/accounts/${aid}/inboxes/${inbox.id}/settings`"
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
          </UPageList>
        </div>
      </div>

      <UModal
        v-model:open="deleteModalOpen"
        :title="t('inboxes.delete.title')"
        :description="t('inboxes.delete.description')"
      >
        <template #body>
          <div class="space-y-3">
            <p class="text-sm text-muted">
              <span class="font-medium text-default">{{ inboxToDelete?.name }}</span>
            </p>
            <UFormField :label="t('inboxes.delete.nameLabel')" name="confirm-name">
              <UInput
                v-model="deleteConfirmName"
                :placeholder="t('inboxes.delete.namePlaceholder')"
                class="w-full"
              />
            </UFormField>
            <p
              v-if="deleteConfirmName && !canConfirmDelete"
              class="text-xs text-error"
            >
              {{ t('inboxes.delete.nameMismatch') }}
            </p>
          </div>
        </template>

        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton
              color="neutral"
              variant="ghost"
              :disabled="deleting"
              @click="closeDeleteModal"
            >
              {{ t('common.cancel') }}
            </UButton>
            <UButton
              color="error"
              :disabled="!canConfirmDelete"
              :loading="deleting"
              @click="confirmDeleteInbox"
            >
              {{ t('common.delete') }}
            </UButton>
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
