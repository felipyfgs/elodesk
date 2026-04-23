import { useConversationsStore, type Conversation } from '~/stores/conversations'
import type { Message } from '~/stores/messages'
import { useMessagesStore } from '~/stores/messages'
import { useAuthStore } from '~/stores/auth'

export function useConversationRealtime(selected: Ref<Conversation | null>) {
  const rt = useRealtime()
  const auth = useAuthStore()
  const convs = useConversationsStore()
  const messages = useMessagesStore()

  function connect() {
    if (auth.account?.id) rt.joinAccount(auth.account.id)

    rt.on<Conversation>('conversation.new', c => convs.upsert(c))
    rt.on<Conversation>('conversation.updated', (c) => {
      convs.upsert(c)
      if (selected.value?.id === c.id) selected.value = c
    })
    rt.on<Message>('message.new', m => messages.upsert(m))
    rt.on<Message>('message.updated', m => messages.upsert(m))
  }

  watch(selected, (c) => {
    if (c) rt.joinConversation(c.id)
  })

  return { connect }
}
