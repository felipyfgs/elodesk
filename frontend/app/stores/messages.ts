import { defineStore } from 'pinia'
import type { Conversation } from './conversations'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export type MessageType = 0 | 1 | 2 | 3
export type MessageStatus = 0 | 1 | 2 | 3 | 'sending'

export type FetchState = 'empty' | 'warmed' | 'fetching' | 'fetched'

export interface MessageAttachmentResponse {
  id: number
  messageId: number
  fileType: string | number
  fileKey?: string
  fileName?: string
  dataUrl?: string
  externalUrl?: string
  extension?: string
  contentType?: string
  size?: number
  createdAt: string | number
}

export interface MessageSender {
  id?: string
  name?: string
  type?: 'contact' | 'user' | 'agent_bot'
  thumbnail?: string | null
  avatarUrl?: string | null
}

export interface Message {
  id: string
  conversationId: string
  inboxId: string
  accountId: string
  content: string | null
  contentType: string
  messageType: MessageType
  sender?: MessageSender | null
  senderType?: 'CONTACT' | 'USER' | 'SYSTEM'
  senderId?: string | null
  sourceId: string | null
  echoId?: string | null
  private?: boolean
  status: MessageStatus
  contentAttributes: Record<string, unknown> | string | null
  forwardedFromMessageId?: number | null
  attachments?: MessageAttachmentResponse[]
  createdAt: string | number
  updatedAt: string | number
}

export type ForwardTarget
  = | { conversationId: string }
    | { contactId: string, inboxId: string }

export interface ForwardResult {
  target: ForwardTarget
  status: 'success' | 'failed'
  createdMessageIds?: number[]
  conversationId?: number
  createdConversation?: boolean
  error?: string
}

export interface ForwardMessagesResponse {
  results: ForwardResult[]
}

export const useMessagesStore = defineStore('messages', {
  state: () => ({
    byConversation: {} as Record<string, Message[]>,
    replyingTo: {} as Record<string, Message | null>,
    fetchState: {} as Record<string, FetchState>,
    fetchedAt: {} as Record<string, number>
  }),
  actions: {
    set(conversationId: string, list: Message[]) {
      this.byConversation[conversationId] = list
      this.fetchState[conversationId] = 'fetched'
      this.fetchedAt[conversationId] = Date.now()
    },
    async fetchMessages(conversationId: string, opts?: { freshMs?: number }) {
      if (!conversationId) return
      if (this.fetchState[conversationId] === 'fetching') return
      const fresh = opts?.freshMs ?? 0
      if (fresh > 0) {
        const last = this.fetchedAt[conversationId] ?? 0
        if (this.fetchState[conversationId] === 'fetched' && last && Date.now() - last < fresh) return
      }
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      const previousState = this.fetchState[conversationId] || 'empty'
      this.fetchState[conversationId] = 'fetching'
      try {
        const res = await api<{ payload: Message[] }>(
          `/accounts/${auth.account.id}/conversations/${conversationId}/messages`
        )
        if (res.payload) {
          this.mergeFetched(conversationId, [...res.payload].reverse())
        }
        this.fetchState[conversationId] = 'fetched'
        this.fetchedAt[conversationId] = Date.now()
      } catch (err) {
        this.fetchState[conversationId] = previousState
        console.error('[messages] fetch failed', err)
      }
    },
    prefetch(conversationId: string) {
      if (!conversationId) return
      const state = this.fetchState[conversationId] || 'empty'
      if (state === 'fetching' || state === 'fetched') return
      void this.fetchMessages(conversationId, { freshMs: 30_000 })
    },
    warmIfEmpty(c: Conversation) {
      if (!c.id || !c.lastNonActivityMessage) return
      const state = this.fetchState[c.id] || 'empty'
      if (state !== 'empty') return
      if ((this.byConversation[c.id]?.length ?? 0) > 0) {
        this.fetchState[c.id] = 'warmed'
        return
      }

      const msg = c.lastNonActivityMessage
      const seeded: Message = {
        id: String(msg.id),
        conversationId: String(c.id),
        inboxId: String(c.inboxId),
        accountId: String(c.accountId),
        content: msg.content ?? null,
        contentType: msg.contentType ?? 'text',
        messageType: msg.messageType as MessageType,
        status: (msg.status ?? 0) as MessageStatus,
        createdAt: msg.createdAt,
        updatedAt: msg.createdAt, // fallback
        private: msg.private,
        sender: msg.sender
          ? {
              id: msg.sender.id,
              name: msg.sender.name,
              type: msg.sender.type,
              thumbnail: msg.sender.thumbnail,
              avatarUrl: msg.sender.avatarUrl
            }
          : null,
        attachments: msg.attachments?.map((a, i) => ({
          id: a.id ?? i,
          messageId: Number(msg.id),
          fileType: a.fileType,
          dataUrl: a.dataUrl,
          externalUrl: a.externalUrl ?? a.fileUrl,
          createdAt: msg.createdAt
        })),
        sourceId: null,
        contentAttributes: {}
      }

      this.byConversation[c.id] = [seeded]
      this.fetchState[c.id] = 'warmed'
    },
    mergeFetched(conversationId: string, list: Message[]) {
      if (!this.byConversation[conversationId]) {
        this.byConversation[conversationId] = list
        this.fetchState[conversationId] = 'fetched'
        return
      }
      for (const m of list) this.upsert(m)
      this.fetchState[conversationId] = 'fetched'
    },
    setReplyTarget(conversationId: string, msg: Message | null) {
      this.replyingTo[conversationId] = msg
    },
    clearReplyTarget(conversationId: string) {
      this.replyingTo[conversationId] = null
    },
    upsert(msg: Message) {
      const bucket = (this.byConversation[String(msg.conversationId)] ||= [])

      if (msg.echoId) {
        const tmpTarget = `tmp:${msg.echoId}`
        const tmpIdx = bucket.findIndex((m) => {
          const id = String(m.id)
          return id === tmpTarget || (id.startsWith('tmp:') && m.echoId === msg.echoId)
        })
        if (tmpIdx >= 0) {
          bucket.splice(tmpIdx, 1)
        }
      }

      const ts = new Date(msg.createdAt).getTime()

      const idx = bucket.findIndex(m => String(m.id) === String(msg.id))
      if (idx >= 0) {
        Object.assign(bucket[idx]!, msg)
        return
      }

      let i = bucket.length - 1
      while (i >= 0 && new Date(bucket[i]!.createdAt).getTime() > ts) i--
      bucket.splice(i + 1, 0, msg)
    },
    remove(id: string | number) {
      const target = String(id)
      for (const convId of Object.keys(this.byConversation)) {
        this.byConversation[convId] = this.byConversation[convId]!.filter(m => String(m.id) !== target)
      }
    },
    async forward({ sourceMessageIds, targets }: { sourceMessageIds: string[], targets: ForwardTarget[] }) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('no account')
      const dtoTargets = targets.map((t) => {
        if ('conversationId' in t) {
          return { conversation_id: Number(t.conversationId) }
        }
        return { contact_id: Number(t.contactId), inbox_id: Number(t.inboxId) }
      })
      return await api<ForwardMessagesResponse>(`/accounts/${auth.account.id}/messages/forward`, {
        method: 'POST',
        body: {
          source_message_ids: sourceMessageIds.map(Number),
          targets: dtoTargets
        }
      })
    }
  }
})
