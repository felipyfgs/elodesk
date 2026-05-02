<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore, type Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useAgentsStore } from '~/stores/agents'
import { forwardSelectionModeKey, forwardSelectedIdsKey } from '~/utils/forward'

const props = withDefaults(defineProps<{
  conversation: Conversation
  showBack?: boolean
}>(), {
  showBack: false
})

const emit = defineEmits<{ close: [] }>()

const api = useApi()
const auth = useAuthStore()
const messages = useMessagesStore()
const conversations = useConversationsStore()
const agents = useAgentsStore()

const { open: detailsOpen } = useDetailsSidebar()
const { isCompact } = useResponsive()

const slideoverOpen = computed({
  get: () => isCompact.value && detailsOpen.value,
  set: (v) => { detailsOpen.value = v }
})

const selectionMode = ref(false)
const selectedMessageIds = ref<Set<string>>(new Set())
const forwardModalOpen = ref(false)

provide(forwardSelectionModeKey, selectionMode)
provide(forwardSelectedIdsKey, selectedMessageIds)

const list = computed<Message[]>(() => messages.byConversation[props.conversation.id] ?? [])

async function markRead(convId: string) {
  if (!auth.account?.id) return
  conversations.markRead(convId)
  try {
    await api(`/accounts/${auth.account.id}/conversations/${convId}/update_last_seen`, { method: 'POST' })
  } catch { void 0 }
}

watch(() => props.conversation.id, async (id) => {
  await messages.fetchMessages(id, { freshMs: 30_000 })
  if (id) markRead(id)
}, { immediate: true })

const scrollContainerRef = ref<HTMLDivElement | null>(null)
const scrollContentRef = ref<HTMLDivElement | null>(null)
const STICK_THRESHOLD_PX = 80
const stickToBottom = ref(true)

const programmaticScroll = ref(false)

function isNearBottom(el: HTMLElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight <= STICK_THRESHOLD_PX
}

function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
  const el = scrollContainerRef.value
  if (!el) return
  programmaticScroll.value = true
  el.scrollTo({ top: el.scrollHeight, behavior })
  const releaseMs = behavior === 'auto' ? 0 : 400
  const release = () => {
    programmaticScroll.value = false
  }
  if (releaseMs === 0) {
    requestAnimationFrame(release)
  } else {
    setTimeout(release, releaseMs)
  }
}

function onScroll() {
  if (programmaticScroll.value) return
  const el = scrollContainerRef.value
  if (!el) return
  stickToBottom.value = isNearBottom(el)
}

const didInitialScroll = ref(false)

function doInitialScroll() {
  if (didInitialScroll.value) return
  didInitialScroll.value = true
  stickToBottom.value = true
  nextTick(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        scrollToBottom('auto')
      })
    })
  })
}

watch(
  () => list.value.length,
  (len) => {
    if (len > 0) doInitialScroll()
  },
  { immediate: true }
)

onMounted(() => {
  const content = scrollContentRef.value
  if (!content) return
  const ro = new ResizeObserver(() => {
    if (!stickToBottom.value) return
    scrollToBottom('auto')
  })
  ro.observe(content)
  onUnmounted(() => ro.disconnect())
})

function cancelSelection() {
  selectionMode.value = false
  selectedMessageIds.value = new Set()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && selectionMode.value) {
    cancelSelection()
  }
}

function onForwardComplete() {
  forwardModalOpen.value = false
  cancelSelection()
}

watch(forwardModalOpen, (open) => {
  if (!open && selectionMode.value) {
    cancelSelection()
  }
})

const selectedMessages = computed<Message[]>(() =>
  list.value.filter(m => selectedMessageIds.value.has(String(m.id)))
)

onMounted(() => {
  if (!agents.items.length) {
    agents.fetch().catch((err) => {
      console.error('[ConversationThread] failed to fetch agents', err)
    })
  }
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <UDashboardPanel id="conversations-thread" class="min-w-0 flex-1">
    <div class="flex min-h-0 flex-1 bg-default">
      <section class="flex min-w-0 flex-1 flex-col bg-default">
        <ConversationsThreadHeader
          v-model:details-open="detailsOpen"
          :conversation="conversation"
          :show-back="showBack"
          @back="emit('close')"
        />

        <div class="flex min-h-0 flex-1 flex-col bg-default">
          <div
            ref="scrollContainerRef"
            class="min-h-0 flex-1 overflow-y-auto px-3 sm:px-4"
            @scroll.passive="onScroll"
          >
            <div ref="scrollContentRef" class="flex min-h-full flex-col justify-end">
              <ConversationsMessageList :messages="list" :conversation="conversation" />
            </div>
          </div>

          <ConversationsSelectionToolbar
            v-if="selectionMode"
            :count="selectedMessageIds.size"
            @cancel="cancelSelection"
            @forward="forwardModalOpen = true"
          />
          <ConversationsComposer v-else :conversation="conversation" />
        </div>
      </section>
      <aside
        v-if="detailsOpen && !isCompact"
        class="hidden w-72 shrink-0 flex-col border-l border-default bg-default lg:flex xl:w-80"
      >
        <ConversationsSidebar
          :conversation="conversation"
          @close="detailsOpen = false"
        />
      </aside>
    </div>
    <USlideover
      v-model:open="slideoverOpen"
      :ui="{ content: 'w-full sm:max-w-sm' }"
    >
      <template #content>
        <ConversationsSidebar
          :conversation="conversation"
          @close="detailsOpen = false"
        />
      </template>
    </USlideover>

    <ConversationsForwardModal
      v-if="selectionMode"
      v-model:open="forwardModalOpen"
      :message-ids="[...selectedMessageIds]"
      :selected-messages="selectedMessages"
      @done="onForwardComplete"
    />
  </UDashboardPanel>
</template>
