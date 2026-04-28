import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

// Backend sends numeric enums; use these helpers/consts for UI mapping.
// MessageType: 0=Incoming, 1=Outgoing, 2=Activity, 3=Template
export type MessageType = 0 | 1 | 2 | 3
// MessageStatus: 0=Sent, 1=Delivered, 2=Read, 3=Failed | 'sending' (optimistic)
export type MessageStatus = 0 | 1 | 2 | 3 | 'sending'

export interface MessageAttachmentResp {
  id: number
  messageId: number
  // Backend sends AttachmentFileType as numeric enum; chatAdapter normalizes.
  fileType: string | number
  fileKey?: string
  // Nome original do arquivo (com acentos/espaços/parênteses) preservado
  // pelo backend separado da chave do MinIO (que é sanitizada).
  fileName?: string
  // URL externa (CDN do Meta/Telegram) usada quando o backend ainda não
  // baixou o blob pro MinIO. Frontend prefere fileKey (signed URL); cai pra
  // externalUrl quando fileKey não existe.
  externalUrl?: string
  extension?: string
  contentType?: string
  size?: number
  // ISO string (optimistic/realtime) or epoch ms (apiAdapter-normalized REST).
  createdAt: string | number
}

// MessageSender mirrors backend dto.MessageSenderResp — polymorphic sender
// embedded in MessageResp. `type` matches Chatwoot's lowercase tokens
// (contact/user/agent_bot); the legacy uppercase `senderType` field below is
// kept for in-memory optimistic messages built by the composer.
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
  // Backend now embeds the polymorphic sender as a struct; senderType/senderId
  // are kept optional for legacy callers and optimistic placeholders.
  sender?: MessageSender | null
  senderType?: 'CONTACT' | 'USER' | 'SYSTEM'
  senderId?: string | null
  sourceId: string | null
  echoId?: string | null
  private?: boolean
  status: MessageStatus
  contentAttributes: Record<string, unknown> | string | null
  forwardedFromMessageId?: number | null
  attachments?: MessageAttachmentResp[]
  // ISO string (optimistic/realtime) or epoch ms (apiAdapter-normalized REST).
  createdAt: string | number
  updatedAt: string | number
}

// Forward types
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

export interface ForwardMessagesResp {
  results: ForwardResult[]
}

export const useMessagesStore = defineStore('messages', {
  state: () => ({
    byConversation: {} as Record<string, Message[]>,
    // Per-conversation reply draft. The composer reads from this to show a
    // quoted preview and embeds the in_reply_to payload on send.
    replyingTo: {} as Record<string, Message | null>
  }),
  actions: {
    set(conversationId: string, list: Message[]) {
      this.byConversation[conversationId] = list
    },
    setReplyTarget(conversationId: string, msg: Message | null) {
      this.replyingTo[conversationId] = msg
    },
    clearReplyTarget(conversationId: string) {
      this.replyingTo[conversationId] = null
    },
    upsert(msg: Message) {
      const bucket = (this.byConversation[String(msg.conversationId)] ||= [])

      // Reconcile optimistic tmp messages by echoId: when the real server
      // message arrives it replaces the `tmp:<echoId>` placeholder.
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

      const idx = bucket.findIndex(m => String(m.id) === String(msg.id))
      if (idx >= 0) bucket[idx] = msg
      else bucket.push(msg)
      // createdAt may arrive as ISO string (optimistic/realtime) or epoch ms
      // (REST responses normalized by apiAdapter). Date(...) handles both.
      bucket.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())
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
      // useApi unwraps the { success, data } envelope, so we get
      // ForwardMessagesResp directly.
      return await api<ForwardMessagesResp>(`/accounts/${auth.account.id}/messages/forward`, {
        method: 'POST',
        body: {
          source_message_ids: sourceMessageIds.map(Number),
          targets: dtoTargets
        }
      })
    }
  }
})
