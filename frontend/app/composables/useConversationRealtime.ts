import { useConversationsStore, type Conversation } from '~/stores/conversations'
import type { Message } from '~/stores/messages'
import { useMessagesStore } from '~/stores/messages'
import { useAuthStore } from '~/stores/auth'

interface ConversationSummaryEvent {
  id: string
  status: number
  assigneeId?: string | null
  teamId?: string | null
  unreadCount?: number
  lastActivityAt: string
}

type MessageWithConversation = Message & { conversation?: ConversationSummaryEvent }

export function useConversationRealtime(selected: Ref<Conversation | null>) {
  const rt = useRealtime()
  const auth = useAuthStore()
  const convs = useConversationsStore()
  const messages = useMessagesStore()

  // Each handler the composable registers is tracked here so we can detach
  // them on unmount — otherwise navigating away and back keeps stacking
  // duplicate listeners on the same event.
  const offHandlers: Array<() => void> = []

  function applyConversationSummary(summary?: ConversationSummaryEvent) {
    if (!summary) return
    const existing = convs.list.find(c => c.id === summary.id)
    if (!existing) return
    const merged: Conversation = {
      ...existing,
      status: summary.status as Conversation['status'],
      assigneeId: summary.assigneeId ?? existing.assigneeId,
      teamId: summary.teamId ?? existing.teamId,
      lastActivityAt: summary.lastActivityAt,
      unreadCount: summary.unreadCount ?? existing.unreadCount
    }
    // selected derives from convs.list/current — upserting is enough to
    // refresh the thread; writing to selected here would re-trigger the
    // route push and is unnecessary.
    convs.upsert(merged)
  }

  function connect() {
    if (auth.account?.id) rt.joinAccount(auth.account.id)
    // Page may have mounted with a conversation already selected (deep link
    // or persisted state) — the watch below only fires on change, so we also
    // join eagerly here.
    if (selected.value?.id) rt.joinConversation(selected.value.id)

    offHandlers.push(rt.on<Conversation>('conversation.created', c => convs.upsert(c)))
    offHandlers.push(rt.on<Conversation>('conversation.updated', c => convs.upsert(c)))
    offHandlers.push(rt.on<MessageWithConversation>('message.created', (m) => {
      messages.upsert(m)
      applyConversationSummary(m.conversation)
    }))
    offHandlers.push(rt.on<MessageWithConversation>('message.updated', (m) => {
      messages.upsert(m)
      applyConversationSummary(m.conversation)
    }))
    offHandlers.push(rt.on<Message>('message.deleted', m => messages.remove(m.id)))
  }

  watch(selected, (c) => {
    if (c) rt.joinConversation(c.id)
  })

  onUnmounted(() => {
    for (const off of offHandlers) off()
    offHandlers.length = 0
  })

  return { connect }
}
