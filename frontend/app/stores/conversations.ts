import { defineStore } from 'pinia'

export interface ConversationContact {
  id: string
  name?: string | null
  phoneNumber?: string | null
  waJid?: string | null
  email?: string | null
  avatarUrl?: string | null
}

export interface ConversationInbox {
  id: string
  name: string
  channelType: string
}

export interface Conversation {
  id: string
  accountId: string
  inboxId: string
  status: 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'
  assigneeId?: string | null
  teamId?: string | null
  contactId: string
  contactInboxId?: string | null
  displayId: number
  uuid: string
  lastActivityAt: string
  additionalAttributes?: string | null
  createdAt: string
  updatedAt: string
  contactInbox?: {
    contact?: ConversationContact
  }
  inbox?: ConversationInbox
  labels?: { id: string, title: string, color: string }[]
  meta?: {
    unreadCount?: number
    assignee?: { name: string, avatarUrl?: string | null }
    sender?: { name: string, avatarUrl?: string | null, thumbnail?: string | null }
    channel?: string
    lastNonActivityMessage?: {
      content: string
      contentType: string
      messageType: string
      createdAt: string
      attachments?: { fileUrl: string, fileType: string }[]
    }
  }
}

export type ConversationTab = 'mine' | 'unassigned' | 'all' | 'mentions'

export interface ConversationFilters {
  tab: ConversationTab
  inboxId?: string
  labelId?: string
  teamId?: string
  status?: string
  from?: string
  to?: string
}

export const useConversationsStore = defineStore('conversations', {
  state: () => ({
    list: [] as Conversation[],
    current: null as Conversation | null,
    loading: false,
    filters: {
      tab: 'mine' as ConversationTab
    } as ConversationFilters,
    selection: [] as string[]
  }),
  getters: {
    filteredList(state): Conversation[] {
      let result = state.list

      // Tab filter — mine/unassigned need the auth store, imported lazily
      if (state.filters.tab === 'unassigned') {
        result = result.filter(c => !c.assigneeId)
      }

      // Status filter
      if (state.filters.status) {
        result = result.filter(c => c.status === state.filters.status)
      }

      // Inbox filter
      if (state.filters.inboxId) {
        result = result.filter(c => c.inboxId === state.filters.inboxId)
      }

      // Label filter
      if (state.filters.labelId) {
        result = result.filter(c => c.labels?.some(l => l.id === state.filters.labelId))
      }

      // Team filter
      if (state.filters.teamId) {
        result = result.filter(c => c.teamId === state.filters.teamId)
      }

      return result
    },
    selectedItems(state): Conversation[] {
      return state.list.filter(c => state.selection.includes(c.id))
    },
    hasSelection(state): boolean {
      return state.selection.length > 0
    }
  },
  actions: {
    setAll(list: Conversation[]) {
      this.list = list
    },
    setCurrent(conv: Conversation | null) {
      this.current = conv
    },
    upsert(conv: Conversation) {
      const idx = this.list.findIndex(c => c.id === conv.id)
      if (idx >= 0) this.list[idx] = conv
      else this.list.unshift(conv)
    },
    remove(id: string) {
      this.list = this.list.filter(c => c.id !== id)
      this.selection = this.selection.filter(sid => sid !== id)
      if (this.current?.id === id) this.current = null
    },
    setFilters(filters: Partial<ConversationFilters>) {
      this.filters = { ...this.filters, ...filters }
    },
    resetFilters() {
      this.filters = { tab: 'mine' }
    },
    toggleSelection(id: string) {
      const idx = this.selection.indexOf(id)
      if (idx >= 0) this.selection.splice(idx, 1)
      else this.selection.push(id)
    },
    selectAll() {
      this.selection = this.filteredList.map(c => c.id)
    },
    clearSelection() {
      this.selection = []
    }
  }
})
