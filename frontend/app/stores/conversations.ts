import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore } from '~/stores/messages'

export interface ConversationInbox {
  id: string
  name: string
  channelType: string
  channelId?: string
  avatarUrl?: string | null
  provider?: string | null
}

export interface ConversationMessageSender {
  id?: string
  name?: string
  type?: 'contact' | 'user' | 'agent_bot'
  thumbnail?: string | null
  avatarUrl?: string | null
}

export interface ConversationLastMessage {
  id?: string
  content?: string | null
  contentType?: string
  messageType: number
  status?: number
  private?: boolean
  createdAt: string | number
  attachments?: { id?: number, dataUrl?: string, externalUrl?: string, fileUrl?: string, fileType: string | number }[]
  sender?: ConversationMessageSender
}

export interface ConversationSender {
  id?: string
  name?: string
  phoneNumber?: string | null
  email?: string | null
  identifier?: string | null
  availabilityStatus?: string
  blocked?: boolean
  avatarUrl?: string | null
  thumbnail?: string | null
  additionalAttributes?: Record<string, unknown> | null
  customAttributes?: Record<string, unknown> | null
}

export interface ConversationAssignee {
  id?: string
  name: string
  email?: string
  avatarUrl?: string | null
  thumbnail?: string | null
}

export interface ConversationTeam {
  id?: string
  name: string
}

export type ConversationStatus = 0 | 1 | 2 | 3

export type ConversationStatusFilter = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'

export const STATUS_MAP: Record<ConversationStatusFilter, ConversationStatus> = {
  OPEN: 0,
  RESOLVED: 1,
  PENDING: 2,
  SNOOZED: 3
}

export const STATUS_CODE: Record<ConversationStatusFilter, string> = {
  OPEN: String(STATUS_MAP.OPEN),
  PENDING: String(STATUS_MAP.PENDING),
  RESOLVED: String(STATUS_MAP.RESOLVED),
  SNOOZED: String(STATUS_MAP.SNOOZED)
}

export interface Conversation {
  id: string
  accountId: string
  inboxId: string
  status: ConversationStatus
  statusName?: string
  assigneeId?: string | null
  teamId?: string | null
  contactId: string
  contactInboxId?: string | null
  displayId: number
  uuid: string
  timestamp?: number
  lastActivityAt: string | number
  firstReplyCreatedAt?: number | null
  agentLastSeenAt?: number | null
  assigneeLastSeenAt?: number | null
  contactLastSeenAt?: number | null
  waitingSince?: number | null
  snoozedUntil?: number | null
  priority?: string | null
  canReply?: boolean
  muted?: boolean
  createdAt: string | number
  updatedAt: string | number
  inbox?: ConversationInbox
  labels?: string[]
  unreadCount?: number
  additionalAttributes?: Record<string, unknown> | null
  customAttributes?: Record<string, unknown> | null
  messages?: ConversationLastMessage[]
  lastNonActivityMessage?: ConversationLastMessage | null
  meta?: {
    sender?: ConversationSender | null
    channel?: string
    assignee?: ConversationAssignee | null
    assigneeType?: string
    team?: ConversationTeam | null
    hmacVerified?: boolean
  }
}

export type ConversationTab = 'mine' | 'all'

export type ConversationType = 'unattended' | 'mention' | 'participating'

export type ConversationSort
  = | 'last_activity_desc'
    | 'last_activity_asc'
    | 'created_desc'
    | 'created_asc'

export interface ConversationFilters {
  tab: ConversationTab
  sortBy: ConversationSort
  inboxIds?: string[]
  labelIds?: string[]
  teamIds?: string[]
  status?: ConversationStatusFilter
  conversationType?: ConversationType
  unread?: boolean
  unassignedOnly?: boolean
  from?: string
  to?: string
}

export interface ConversationMetaBucket {
  all: number
  mine: number
  unassigned: number
  unread: number
}

export interface ConversationMeta {
  open: ConversationMetaBucket
  pending: ConversationMetaBucket
  resolved: ConversationMetaBucket
  snoozed: ConversationMetaBucket
}

export interface ConversationListMeta {
  mineCount: number
  assignedCount: number
  unassignedCount: number
  allCount: number
}

export interface ConversationListResponse {
  meta: ConversationListMeta
  payload: Conversation[]
}

const EMPTY_BUCKET: ConversationMetaBucket = { all: 0, mine: 0, unassigned: 0, unread: 0 }

function emptyMeta(): ConversationMeta {
  return {
    open: { ...EMPTY_BUCKET },
    pending: { ...EMPTY_BUCKET },
    resolved: { ...EMPTY_BUCKET },
    snoozed: { ...EMPTY_BUCKET }
  }
}

function emptyListMeta(): ConversationListMeta {
  return { mineCount: 0, assignedCount: 0, unassignedCount: 0, allCount: 0 }
}

export const useConversationsStore = defineStore('conversations', {
  state: () => ({
    list: [] as Conversation[],
    current: null as Conversation | null,
    isLoading: false,
    filters: {
      tab: 'mine' as ConversationTab,
      sortBy: 'last_activity_desc' as ConversationSort,
      status: 'OPEN' as ConversationStatusFilter
    } as ConversationFilters,
    meta: emptyMeta() as ConversationMeta,
    listMeta: emptyListMeta() as ConversationListMeta,
    selection: [] as string[],
    stickyUnreadId: null as string | null,
    manuallyUnread: [] as string[]
  }),
  getters: {
    filteredList(state): Conversation[] {
      let result: Conversation[] = Array.isArray(state.list) ? state.list : []

      if (state.filters.tab === 'mine') {
        const auth = useAuthStore()
        const myId = auth.user?.id
        if (myId) {
          result = result.filter(c => String(c.assigneeId) === String(myId))
        }
      }
      if (state.filters.unassignedOnly) {
        result = result.filter(c => !c.assigneeId)
      }

      if (state.filters.status) {
        const statusNum = STATUS_MAP[state.filters.status]
        result = result.filter(c => c.status === statusNum)
      }

      const inboxIds = state.filters.inboxIds
      if (inboxIds && inboxIds.length) {
        const set = new Set(inboxIds.map(String))
        result = result.filter(c => set.has(String(c.inboxId)))
      }

      const labelIds = state.filters.labelIds
      if (labelIds && labelIds.length) {
        const set = new Set(labelIds)
        result = result.filter(c => c.labels?.some(l => set.has(l)))
      }

      const teamIds = state.filters.teamIds
      if (teamIds && teamIds.length) {
        const set = new Set(teamIds.map(String))
        result = result.filter(c => set.has(String(c.teamId)))
      }

      if (state.filters.conversationType === 'unattended') {
        result = result.filter(c => !c.firstReplyCreatedAt || !!c.waitingSince)
      }

      if (state.filters.unread) {
        const sticky = state.stickyUnreadId
        const manualSet = new Set(state.manuallyUnread)
        result = result.filter(c => (c.unreadCount ?? 0) > 0 || c.id === sticky || manualSet.has(c.id))
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
      const messages = useMessagesStore()
      for (const c of this.list) messages.warmIfEmpty(c)
    },
    setCurrent(conv: Conversation | null) {
      this.current = conv
      if (this.stickyUnreadId && conv?.id !== this.stickyUnreadId) {
        this.stickyUnreadId = null
      }
    },
    markRead(id: string) {
      const muIdx = this.manuallyUnread.indexOf(id)
      const wasManual = muIdx >= 0
      if (wasManual) this.manuallyUnread.splice(muIdx, 1)

      const conv = this.list.find(c => c.id === id) ?? (this.current?.id === id ? this.current : null)
      if (!conv) return
      const hadRealUnread = (conv.unreadCount ?? 0) > 0
      if (hadRealUnread || wasManual) {
        this.stickyUnreadId = id
      }
      if (hadRealUnread) {
        this.upsert({ ...conv, unreadCount: 0 })
        this.bumpMetaUnread(conv.status, -1)
      }
    },
    bumpMetaUnread(status: ConversationStatus, delta: number) {
      const map: Record<ConversationStatus, keyof ConversationMeta> = {
        0: 'open', 1: 'resolved', 2: 'pending', 3: 'snoozed'
      }
      const k = map[status]
      const bucket = this.meta[k]
      if (!bucket) return
      bucket.unread = Math.max(0, bucket.unread + delta)
    },
    markAsUnread(id: string) {
      if (!this.manuallyUnread.includes(id)) {
        this.manuallyUnread.push(id)
      }
      if (this.stickyUnreadId === id) this.stickyUnreadId = null
    },
    setManuallyUnread(ids: string[]) {
      this.manuallyUnread = Array.isArray(ids) ? [...ids] : []
    },
    upsert(conv: Conversation) {
      const idx = this.list.findIndex(c => c.id === conv.id)
      if (idx >= 0) this.list[idx] = conv
      else this.list.unshift(conv)
      if (this.current?.id === conv.id) this.current = conv

      useMessagesStore().warmIfEmpty(conv)
    },
    applyPatch(id: string, patch: Partial<Conversation>) {
      const idStr = String(id)
      const idx = this.list.findIndex(c => String(c.id) === idStr)
      if (idx >= 0) this.list[idx] = { ...this.list[idx]!, ...patch }
      if (this.current && String(this.current.id) === idStr) {
        this.current = { ...this.current, ...patch }
      }
    },
    remove(id: string) {
      this.list = this.list.filter(c => c.id !== id)
      this.selection = this.selection.filter(sid => sid !== id)
      if (this.current?.id === id) this.current = null
    },
    setFilters(filters: Partial<ConversationFilters>) {
      this.filters = { ...this.filters, ...filters }
      this.stickyUnreadId = null
    },
    resetFilters() {
      this.filters = { tab: 'mine', sortBy: 'last_activity_desc', status: 'OPEN' }
      this.stickyUnreadId = null
    },
    clearScopeFilters() {
      this.filters = { ...this.filters, inboxIds: undefined, labelIds: undefined, teamIds: undefined }
      this.stickyUnreadId = null
    },
    setMeta(meta: ConversationMeta) {
      this.meta = meta
    },
    setListMeta(meta: ConversationListMeta) {
      this.listMeta = meta
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
