<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'

const isOpen = defineModel<boolean>('open', { default: false })
const selectedInboxId = defineModel<string | undefined>('selectedInboxId')

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const inboxesStore = useInboxesStore()
const errorHandler = useErrorHandler()

const loadingInboxes = ref(false)

const selectedInbox = computed(() =>
  inboxesStore.list.find(i => i.id === selectedInboxId.value) ?? null
)

const inboxItems = computed(() =>
  inboxesStore.list.map(i => ({
    label: i.name,
    icon: channelIcon(i.channelType),
    value: i.id
  }))
)

async function loadInboxes() {
  if (inboxesStore.list.length > 0 || !auth.account?.id) return
  loadingInboxes.value = true
  try {
    const res = await api<Inbox[]>(`/accounts/${auth.account.id}/inboxes`)
    inboxesStore.setAll(res)
  } catch (err) {
    errorHandler.handle(err)
  } finally {
    loadingInboxes.value = false
  }
}

watch(isOpen, async (open) => {
  if (open) await loadInboxes()
})

function clear() {
  selectedInboxId.value = undefined
}
</script>

<template>
  <div class="flex items-center gap-3 px-4 py-2.5 min-h-11">
    <span class="text-sm font-medium text-muted shrink-0 w-10">
      {{ t('contactsSendMessage.viaLabel') }}:
    </span>
    <UBadge
      v-if="selectedInbox"
      color="primary"
      variant="soft"
      size="md"
    >
      <UIcon :name="channelIcon(selectedInbox.channelType)" class="size-3.5 mr-1" />
      <span class="truncate max-w-[14rem]">{{ selectedInbox.name }}</span>
      <UButton
        icon="i-lucide-x"
        variant="link"
        color="neutral"
        size="xs"
        :padded="false"
        class="ml-1 p-0"
        @click="clear"
      />
    </UBadge>
    <USelectMenu
      v-else
      v-model="selectedInboxId"
      :items="inboxItems"
      value-key="value"
      :placeholder="t('contactsSendMessage.viaPlaceholder')"
      :loading="loadingInboxes"
      variant="ghost"
      class="flex-1"
    />
  </div>
</template>
