import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'

// Conversation shape mirrors Chatwoot's `_conversation.json.jbuilder` (see
// backend/internal/dto/conversation.go::ConversationResp). Field names below
// are camelCase because `utils/apiAdapter.ts` rewrites every snake_case key on
// response. Anything that was nested under `contactInbox.contact` now lives at
// `meta.sender`; the legacy field has been removed.

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
  attachments?: { fileUrl?: string, fileType: string | number }[]
  sender?: ConversationMessageSender
}

// ConversationSender mirrors backend ContactResp: it's the contact who started
// the conversation, embedded in `meta.sender`.
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

export type ConversationTab = 'mine' | 'unassigned' | 'all'

export type ConversationSort
  = | 'last_activity_desc'
    | 'last_activity_asc'
    | 'created_desc'
    | 'created_asc'

// `ALL` é uma virtualidade do frontend — quando selecionado, o filtro `status`
// não vai pra API e o backend devolve conversas em qualquer status. Mantém a
// fonte única no STATUS_MAP que mapeia direto pros enums numéricos do backend.
export type ConversationStatusFilter = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED' | 'ALL'

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

// Legacy meta envelope returned by GET /conversations/meta — open/pending/resolved/snoozed
// × all/mine/unassigned. Kept until the dashboard tab counters move to the new
// flat shape (ConversationListMeta).
export interface ConversationMeta {
  open: ConversationMetaBucket
  pending: ConversationMetaBucket
  resolved: ConversationMetaBucket
  snoozed: ConversationMetaBucket
}

// New Chatwoot-shape meta returned alongside the conversations list payload.
// Mirrors backend dto.ConversationListMeta.
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

const EMPTY_BUCKET: ConversationMetaBucket = { all: 0, mine: 0, unassigned: 0 }

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
    loading: false,
    filters: {
      tab: 'mine' as ConversationTab,
      sortBy: 'last_activity_desc' as ConversationSort,
      status: 'OPEN' as ConversationStatusFilter
    } as ConversationFilters,
    meta: emptyMeta() as ConversationMeta,
    listMeta: emptyListMeta() as ConversationListMeta,
    selection: [] as string[]
  }),
  getters: {
    filteredList(state): Conversation[] {
      let result: Conversation[] = Array.isArray(state.list) ? state.list : []

      // Tab filter — mine/unassigned need the auth store, imported lazily.
      // IDs are coerced via String() because backend sends int64 (JS number)
      // while stores/filters keep them as strings. `1 === "1"` would be false.
      if (state.filters.tab === 'unassigned') {
        result = result.filter(c => !c.assigneeId)
      } else if (state.filters.tab === 'mine') {
        const auth = useAuthStore()
        const myId = auth.user?.id
        if (myId) {
          result = result.filter(c => String(c.assigneeId) === String(myId))
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
        result = result.filter(c => String(c.inboxId) === String(state.filters.inboxId))
      }

      // Label filter — backend now sends labels as string titles (Chatwoot
      // shape). The legacy object form is no longer surfaced here, so the
      // filterId is interpreted as the label title for backward compat with
      // saved filters that still reference an id.
      if (state.filters.labelId) {
        result = result.filter(c => c.labels?.includes(state.filters.labelId as string))
      }

      // Team filter
      if (state.filters.teamId) {
        result = result.filter(c => String(c.teamId) === String(state.filters.teamId))
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
      // `current` is a separate reference from `list[idx]`. Without this,
      // ConversationsIndex's `selected` computed (which reads `current` first)
      // returns the stale object after upsert, so child props don't update.
      if (this.current?.id === conv.id) this.current = conv
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
