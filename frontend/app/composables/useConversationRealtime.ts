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

  // Each handler the composable registers is tracked here so we can detach
  // them on unmount — otherwise navigating away and back keeps stacking
  // duplicate listeners on the same event.
  const offHandlers: Array<() => void> = []

  // Debounce meta refresh so a burst of realtime events (e.g. bulk resolve)
  // collapses into a single HTTP roundtrip. Counts in convs.meta come from a
  // dedicated /meta endpoint; we don't try to derive them client-side because
  // the loaded list is always status-scoped and can't see other buckets.
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
    // applyPatch atualiza a conversa onde quer que ela esteja (list e/ou
    // current) sem inseri-la em listas filtradas que não a contêm. Antes,
    // chamar `convs.upsert` aqui injetava a conversa no topo da lista mesmo
    // quando o filtro não devia mostrá-la — quebrava a aba "Resolvidas" se
    // chegasse uma mensagem nova na conversa Open aberta.
    //
    // O summary embutido em message.created/updated usa `omitempty` no backend
    // para `assigneeId`/`teamId`, ou seja, ausência ≠ "nulo". Por isso só
    // patcheamos campos efetivamente presentes — preserva o assignee/team
    // atual se a mensagem não trouxe o dado.
    const patch: Partial<Conversation> = {
      status: summary.status as Conversation['status'],
      lastActivityAt: summary.lastActivityAt
    }
    if (summary.assigneeId !== undefined) patch.assigneeId = summary.assigneeId
    if (summary.teamId !== undefined) patch.teamId = summary.teamId
    if (summary.unreadCount !== undefined) patch.unreadCount = summary.unreadCount
    convs.applyPatch(String(summary.id), patch)
  }

  // Atualiza a prévia (lastNonActivityMessage) do card da lista quando uma
  // mensagem não-atividade/template/privada chega via realtime. Antes só
  // mexíamos em status/unread/lastActivityAt — a prévia ficava congelada
  // até o próximo refetch (ex: trocar de tab).
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
    // Page may have mounted with a conversation already selected (deep link
    // or persisted state) — the watch below only fires on change, so we also
    // join eagerly here.
    if (selected.value?.id) rt.joinConversation(selected.value.id)

    offHandlers.push(rt.on<Conversation>('conversation.created', (c) => {
      // convs.upsert also triggers messages.warmIfEmpty(c)
      convs.upsert(c)
      invalidateMeta()
    }))
    offHandlers.push(rt.on<Conversation>('conversation.updated', (c) => {
      // convs.upsert also triggers messages.warmIfEmpty(c)
      convs.upsert(c)
      invalidateMeta()
    }))
    // Backend emits numeric IDs (int64 → JSON number) and the store also holds
    // numeric IDs at runtime even though TS types claim string. Pass-through to
    // remove() so strict equality matches; coercing here would silently no-op.
    offHandlers.push(rt.on<{ id: string }>('conversation.deleted', (e) => {
      convs.remove(e.id)
      invalidateMeta()
    }))
    offHandlers.push(rt.on<MessageWithConversation>('message.created', (m) => {
      // messages.upsert(m) seeds the bucket; no extra warmup state change
      // needed here as the next prefetch/fetch will see 'empty'/'warmed'
      // and pull history.
      messages.upsert(m)
      applyConversationSummary(m.conversation)
      applyLastMessage(m)
      // message.created can flip status (reopen) and bumps unread → meta refresh.
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
