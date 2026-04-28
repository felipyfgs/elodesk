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

const detailsOpen = ref(true)

// Forward selection state
const selectionMode = ref(false)
const selectedMessageIds = ref<Set<string>>(new Set())
const forwardModalOpen = ref(false)

provide(forwardSelectionModeKey, selectionMode)
provide(forwardSelectedIdsKey, selectedMessageIds)

const list = computed<Message[]>(() => messages.byConversation[props.conversation.id] ?? [])

async function loadMessages() {
  if (!auth.account?.id) return
  const res = await api<{ payload: Message[] }>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`)
  if (res.payload) {
    messages.set(props.conversation.id, [...res.payload].reverse())
  }
}

// Marca a conversa como lida assim que o agente abre. Optimistic: zera o
// unreadCount localmente e chama o endpoint que atualiza assignee_last_seen_at
// e dispara conversation.updated para todos os clientes. Best-effort: erros
// de rede são silenciados — o badge volta na próxima hidratação.
async function markRead(convId: string) {
  if (!auth.account?.id) return
  if ((props.conversation.unreadCount ?? 0) > 0) {
    conversations.upsert({ ...props.conversation, unreadCount: 0 })
  }
  try {
    await api(`/accounts/${auth.account.id}/conversations/${convId}/update_last_seen`, { method: 'POST' })
  } catch {
    // ignore — broadcast realtime will reconcile if/when it succeeds
  }
}

watch(() => props.conversation.id, async (id) => {
  await loadMessages()
  if (id) markRead(id)
}, { immediate: true })

// Scroll-to-bottom: handled at the outer overflow container instead of
// trusting UChatMessages :auto-scroll. Reactive list updates from the
// realtime store don't reliably trigger Nuxt UI's internal observer when
// the scroll element is the parent overflow div.
const scrollContainerRef = ref<HTMLDivElement | null>(null)
const STICK_THRESHOLD_PX = 80
const stickToBottom = ref(true)

function isNearBottom(el: HTMLElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight <= STICK_THRESHOLD_PX
}

function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
  const el = scrollContainerRef.value
  if (!el) return
  el.scrollTo({ top: el.scrollHeight, behavior })
}

function onScroll() {
  const el = scrollContainerRef.value
  if (!el) return
  // If the agent reads history (scrolls up), don't yank them back when a
  // new message arrives. Resume auto-stick once they're back near the end.
  stickToBottom.value = isNearBottom(el)
}

// Reset to bottom whenever the active conversation changes.
watch(() => props.conversation.id, () => {
  stickToBottom.value = true
  nextTick(() => scrollToBottom('auto'))
})

// React to new/updated messages. Watching length covers append; watching
// the last id covers the "pending → real" reconciliation in upsert.
watch(
  () => [list.value.length, list.value[list.value.length - 1]?.id] as const,
  () => {
    if (!stickToBottom.value) return
    nextTick(() => scrollToBottom('smooth'))
  }
)

// Cancel selection mode and clear
function cancelSelection() {
  selectionMode.value = false
  selectedMessageIds.value = new Set()
}

// Handle Esc key to exit selection mode
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && selectionMode.value) {
    cancelSelection()
  }
}

// Handler for when forward completes (success)
function onForwardComplete() {
  forwardModalOpen.value = false
  cancelSelection()
}

// When the modal is dismissed without submitting (cancel/click-outside),
// exit selection mode too.
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
        <ConversationsSelectionToolbar
          v-if="selectionMode"
          :count="selectedMessageIds.size"
          @cancel="cancelSelection"
          @forward="forwardModalOpen = true"
        />
        <ConversationsThreadHeader
          v-else
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
            <ConversationsMessageList :messages="list" :conversation="conversation" />
          </div>

          <ConversationsComposer :conversation="conversation" />
        </div>
      </section>

      <ConversationsSidebar
        v-if="detailsOpen"
        :conversation="conversation"
        @close="detailsOpen = false"
      />
    </div>

    <ConversationsForwardModal
      v-if="selectionMode"
      v-model:open="forwardModalOpen"
      :message-ids="[...selectedMessageIds]"
      :selected-messages="selectedMessages"
      @done="onForwardComplete"
    />
  </UDashboardPanel>
</template>
