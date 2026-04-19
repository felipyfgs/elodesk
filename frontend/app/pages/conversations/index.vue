<script setup lang="ts">
import { breakpointsTailwind } from '@vueuse/core'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import type { Message } from '~/stores/messages'
import { useMessagesStore } from '~/stores/messages'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const rt = useRealtime()
const auth = useAuthStore()
const convs = useConversationsStore()
const messages = useMessagesStore()

const tabItems = computed(() => [
  { label: t('conversations.filters.all'), value: 'all' },
  { label: t('conversations.filters.open'), value: 'OPEN' },
  { label: t('conversations.filters.pending'), value: 'PENDING' },
  { label: t('conversations.filters.resolved'), value: 'RESOLVED' }
])
const selectedTab = ref('all')

const filtered = computed<Conversation[]>(() => {
  if (selectedTab.value === 'all') return convs.list
  return convs.list.filter(c => c.status === selectedTab.value)
})

const selected = ref<Conversation | null>(null)
const isPanelOpen = computed({
  get: () => !!selected.value,
  set: (v: boolean) => { if (!v) selected.value = null }
})

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('lg')

async function load() {
  if (!auth.account?.id) return
  const list = await api<Conversation[]>(`/accounts/${auth.account.id}/conversations`)
  convs.setAll(list)
}

onMounted(async () => {
  await load()
  if (auth.account?.id) rt.joinAccount(auth.account.id)
  rt.on<Conversation>('conversation.new', c => convs.upsert(c))
  rt.on<Conversation>('conversation.updated', c => convs.upsert(c))
  rt.on<Message>('message.new', m => messages.upsert(m))
  rt.on<Message>('message.updated', m => messages.upsert(m))
})

watch(selected, (c) => {
  if (c) rt.joinConversation(c.id)
})

watch(filtered, () => {
  if (!filtered.value.find(c => c.id === selected.value?.id)) {
    selected.value = null
  }
})
</script>

<template>
  <UDashboardPanel
    id="conv-list"
    :default-size="25"
    :min-size="20"
    :max-size="35"
    resizable
  >
    <UDashboardNavbar :title="t('conversations.title')">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
      <template #trailing>
        <UBadge :label="filtered.length" variant="subtle" />
      </template>
      <template #right>
        <UTabs
          v-model="selectedTab"
          :items="tabItems"
          :content="false"
          size="xs"
        />
      </template>
    </UDashboardNavbar>

    <ConversationsList v-model="selected" :items="filtered" />
  </UDashboardPanel>

  <ConversationsThread
    v-if="selected && !isMobile"
    :conversation="selected"
    @close="selected = null"
  />
  <div v-else-if="!isMobile" class="hidden lg:flex flex-1 items-center justify-center flex-col gap-2">
    <UIcon name="i-lucide-inbox" class="size-32 text-dimmed" />
    <p class="text-muted">
      {{ t('conversations.select') }}
    </p>
  </div>

  <ClientOnly>
    <USlideover v-if="isMobile" v-model:open="isPanelOpen">
      <template #content>
        <ConversationsThread
          v-if="selected"
          :conversation="selected"
          @close="selected = null"
        />
      </template>
    </USlideover>
  </ClientOnly>
</template>
