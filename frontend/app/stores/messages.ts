import { defineStore } from 'pinia'

// Backend sends numeric enums; use these helpers/consts for UI mapping.
// MessageType: 0=Incoming, 1=Outgoing, 2=Activity, 3=Template
export type MessageType = 0 | 1 | 2 | 3
// MessageStatus: 0=Sent, 1=Delivered, 2=Read, 3=Failed
export type MessageStatus = 0 | 1 | 2 | 3

export interface Message {
  id: string
  conversationId: string
  inboxId: string
  accountId: string
  content: string | null
  contentType: string
  messageType: MessageType
  senderType: 'CONTACT' | 'USER' | 'SYSTEM'
  senderId: string | null
  sourceId: string | null
  private?: boolean
  status: MessageStatus
  contentAttributes: Record<string, unknown> | string | null
  createdAt: string
  updatedAt: string
}

export const useMessagesStore = defineStore('messages', {
  state: () => ({
    byConversation: {} as Record<string, Message[]>
  }),
  actions: {
    set(conversationId: string, list: Message[]) {
      this.byConversation[conversationId] = list
    },
    upsert(msg: Message) {
      const bucket = (this.byConversation[msg.conversationId] ||= [])
      const idx = bucket.findIndex(m => m.id === msg.id)
      if (idx >= 0) bucket[idx] = msg
      else bucket.push(msg)
      bucket.sort((a, b) => a.createdAt.localeCompare(b.createdAt))
    }
  }
})
