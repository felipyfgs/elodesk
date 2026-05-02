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

export function useConversationRealtime(
  selected: Ref<Conversation | null>,
  onMetaInvalidate?: () => void
) {
  const rt = useRealtime()
  const auth = useAuthStore()
  const convs = useConversationsStore()
  const messages = useMessagesStore()

  const offHandlers: Array<() => void> = []

  let metaTimer: ReturnType<typeof setTimeout> | null = null
  function invalidateMeta() {
    if (!onMetaInvalidate) return
    if (metaTimer) clearTimeout(metaTimer)
    metaTimer = setTimeout(() => {
      metaTimer = null
      onMetaInvalidate()
    }, 200)
  }

  function applyConversationSummary(summary?: ConversationSummaryEvent) {
    if (!summary) return
    const patch: Partial<Conversation> = {
      status: summary.status as Conversation['status'],
      lastActivityAt: summary.lastActivityAt
    }
    if (summary.assigneeId !== undefined) patch.assigneeId = summary.assigneeId
    if (summary.teamId !== undefined) patch.teamId = summary.teamId
    if (summary.unreadCount !== undefined) patch.unreadCount = summary.unreadCount
    convs.applyPatch(String(summary.id), patch)
  }

  function applyLastMessage(m: Message) {
    if (m.messageType === 2 || m.messageType === 3) return
    if (m.private) return
    const existing = convs.list.find(c => c.id === m.conversationId)
    if (!existing) return
    const lnam: NonNullable<Conversation['lastNonActivityMessage']> = {
      id: m.id,
      content: m.content,
      contentType: m.contentType,
      messageType: m.messageType,
      status: typeof m.status === 'number' ? m.status : undefined,
      private: m.private,
      createdAt: m.createdAt,
      attachments: m.attachments?.map(a => ({ fileType: a.fileType })),
      sender: m.sender
        ? {
            id: m.sender.id,
            name: m.sender.name,
            type: m.sender.type,
            thumbnail: m.sender.thumbnail,
            avatarUrl: m.sender.avatarUrl
          }
        : undefined
    }
    convs.upsert({ ...existing, lastNonActivityMessage: lnam })
  }

  function connect() {
    if (auth.account?.id) rt.joinAccount(auth.account.id)
    if (selected.value?.id) rt.joinConversation(selected.value.id)

    offHandlers.push(rt.on<Conversation>('conversation.created', (c) => {
      convs.upsert(c)
      invalidateMeta()
    }))
    offHandlers.push(rt.on<Conversation>('conversation.updated', (c) => {
      convs.upsert(c)
      invalidateMeta()
    }))
    offHandlers.push(rt.on<{ id: string }>('conversation.deleted', (e) => {
      convs.remove(e.id)
      invalidateMeta()
    }))
    offHandlers.push(rt.on<MessageWithConversation>('message.created', (m) => {
      messages.upsert(m)
      applyConversationSummary(m.conversation)
      applyLastMessage(m)
      invalidateMeta()
    }))
    offHandlers.push(rt.on<MessageWithConversation>('message.updated', (m) => {
      messages.upsert(m)
      applyConversationSummary(m.conversation)
      applyLastMessage(m)
    }))
    offHandlers.push(rt.on<Message>('message.deleted', m => messages.remove(m.id)))
  }

  watch(selected, (c) => {
    if (c) rt.joinConversation(c.id)
  })

  onUnmounted(() => {
    for (const off of offHandlers) off()
    offHandlers.length = 0
    if (metaTimer) {
      clearTimeout(metaTimer)
      metaTimer = null
    }
  })

  return { connect }
}
