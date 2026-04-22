import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'

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

// Backend sends status as int: 0=Open, 1=Resolved, 2=Pending, 3=Snoozed
export type ConversationStatus = 0 | 1 | 2 | 3

export const STATUS_MAP: Record<string, ConversationStatus> = {
  OPEN: 0,
  RESOLVED: 1,
  PENDING: 2,
  SNOOZED: 3
}

export interface Conversation {
  id: string
  accountId: string
  inboxId: string
  status: ConversationStatus
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

export type ConversationSort
  = | 'last_activity_desc'
    | 'last_activity_asc'
    | 'created_desc'
    | 'created_asc'

export type ConversationStatusFilter = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'

export interface ConversationFilters {
  tab: ConversationTab
  sortBy: ConversationSort
  inboxId?: string
  labelId?: string
  teamId?: string
  status?: ConversationStatusFilter
  from?: string
  to?: string
}

export interface ConversationMetaBucket {
  all: number
  mine: number
  unassigned: number
}

export interface ConversationMeta {
  open: ConversationMetaBucket
  pending: ConversationMetaBucket
  resolved: ConversationMetaBucket
  snoozed: ConversationMetaBucket
}

const EMPTY_BUCKET: ConversationMetaBucket = { all: 0, mine: 0, unassigned: 0 }

function emptyMeta(): ConversationMeta {
  return {
    open: { ...EMPTY_BUCKET },
    pending: { ...EMPTY_BUCKET },
    resolved: { ...EMPTY_BUCKET },
    snoozed: { ...EMPTY_BUCKET }
  }
}

export const useConversationsStore = defineStore('conversations', {
  state: () => ({
    list: [] as Conversation[],
    current: null as Conversation | null,
    loading: false,
    filters: {
      tab: 'mine' as ConversationTab,
      sortBy: 'last_activity_desc' as ConversationSort,
      status: 'OPEN' as ConversationStatusFilter
    } as ConversationFilters,
    meta: emptyMeta() as ConversationMeta,
    selection: [] as string[]
  }),
  getters: {
    filteredList(state): Conversation[] {
      let result: Conversation[] = Array.isArray(state.list) ? state.list : []

      // Tab filter — mine/unassigned need the auth store, imported lazily
      if (state.filters.tab === 'unassigned') {
        result = result.filter(c => !c.assigneeId)
      } else if (state.filters.tab === 'mine') {
        const auth = useAuthStore()
        const myId = auth.user?.id
        if (myId) {
          result = result.filter(c => c.assigneeId === myId)
        }
      }

      // Status filter — backend sends int, filters are string labels
      if (state.filters.status) {
        const statusNum = STATUS_MAP[state.filters.status]
        if (statusNum !== undefined) {
          result = result.filter(c => c.status === statusNum)
        }
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
      this.list = Array.isArray(list) ? list : []
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
      this.filters = { tab: 'mine', sortBy: 'last_activity_desc', status: 'OPEN' }
    },
    setMeta(meta: ConversationMeta) {
      this.meta = meta
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
