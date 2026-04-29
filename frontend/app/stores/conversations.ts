import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore } from '~/stores/messages'

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

// Filtros aceitos no estado: os 4 statuses do backend. "Todos" é representado
// por `filters.status === undefined` — sem param na API, sem clausura
// client-side. UI traduz isso pra um item visível ("Todas") no dropdown.
export type ConversationStatusFilter = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'

export const STATUS_MAP: Record<ConversationStatusFilter, ConversationStatus> = {
  OPEN: 0,
  RESOLVED: 1,
  PENDING: 2,
  SNOOZED: 3
}

// String form usada pelo querystring da API (`status=0`). Derivado direto do
// STATUS_MAP pra evitar drift — fonte única dessa tabela.
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

// `tab` é a dimensão do tab visível na UI. "Sem agente" virou flag separada
// (ConversationFilters.unassignedOnly) pra não ser apagada ao trocar de tab.
export type ConversationTab = 'mine' | 'all'

// Filtros ortogonais à tab. `unattended` (Chatwoot): conversas em que o cliente
// está esperando resposta — `firstReplyCreatedAt IS NULL` OU `waitingSince IS
// NOT NULL`. Pode coexistir com qualquer tab (ex.: "minhas não atendidas").
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
  // Conversas com mensagens não-lidas dentro desse status. Backend computa
  // global (não particiona por assignee — o badge "Não lidas" é sempre
  // global, decisão de produto).
  unread: number
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
    loading: false,
    filters: {
      tab: 'mine' as ConversationTab,
      sortBy: 'last_activity_desc' as ConversationSort,
      status: 'OPEN' as ConversationStatusFilter
    } as ConversationFilters,
    meta: emptyMeta() as ConversationMeta,
    listMeta: emptyListMeta() as ConversationListMeta,
    selection: [] as string[],
    // Conversa atualmente aberta cujo unreadCount foi zerado pela leitura.
    // Mantém ela visível na aba "Não lidas" enquanto estiver selecionada —
    // só sai da lista quando o agente troca de conversa ou fecha.
    stickyUnreadId: null as string | null,
    // IDs de conversas marcadas manualmente como "não lidas" pelo agente
    // (right-click → Marcar como não lida). Espelha o comportamento do
    // WhatsApp Web: é puramente local — não vai pro backend, não muda
    // assignee_last_seen_at, não afeta read receipts. Persistido em
    // localStorage por `useConversationFilters` na entrada da página.
    manuallyUnread: [] as string[]
  }),
  getters: {
    filteredList(state): Conversation[] {
      let result: Conversation[] = Array.isArray(state.list) ? state.list : []

      // Tab filter — `mine` need the auth store, imported lazily. IDs are
      // coerced via String() because backend sends int64 (JS number) while
      // stores/filters keep them as strings. `1 === "1"` would be false.
      if (state.filters.tab === 'mine') {
        const auth = useAuthStore()
        const myId = auth.user?.id
        if (myId) {
          result = result.filter(c => String(c.assigneeId) === String(myId))
        }
      }
      // Independent "Sem agente" flag — intersects with any tab. `mine` +
      // `unassignedOnly` results in an empty list (correct: contradictory).
      if (state.filters.unassignedOnly) {
        result = result.filter(c => !c.assigneeId)
      }

      // Status filter — backend sends int, filters are string labels.
      // `undefined` significa "todos os status" e desliga esse passo.
      if (state.filters.status) {
        const statusNum = STATUS_MAP[state.filters.status]
        result = result.filter(c => c.status === statusNum)
      }

      // Inbox filter (multi-select)
      const inboxIds = state.filters.inboxIds
      if (inboxIds && inboxIds.length) {
        const set = new Set(inboxIds.map(String))
        result = result.filter(c => set.has(String(c.inboxId)))
      }

      // Label filter (multi-select). Backend sends labels as string titles
      // (Chatwoot shape); a conversation matches if it carries any of the
      // selected labels.
      const labelIds = state.filters.labelIds
      if (labelIds && labelIds.length) {
        const set = new Set(labelIds)
        result = result.filter(c => c.labels?.some(l => set.has(l)))
      }

      // Team filter (multi-select)
      const teamIds = state.filters.teamIds
      if (teamIds && teamIds.length) {
        const set = new Set(teamIds.map(String))
        result = result.filter(c => set.has(String(c.teamId)))
      }

      // Conversation type — ortogonal às outras dimensões. Espelha o scope
      // `unattended` do Chatwoot (`models/conversation.rb`): primeira resposta
      // ainda não enviada OU cliente esperando resposta agora.
      if (state.filters.conversationType === 'unattended') {
        result = result.filter(c => !c.firstReplyCreatedAt || !!c.waitingSince)
      }

      // Unread — `unreadCount > 0`. Ortogonal: combina com qualquer status,
      // assignee tab, etc. Não vai pro backend (sem param dedicado), o filtro
      // é só client-side em cima da página carregada. Também passam:
      // - sticky (conversa aberta agora, recém-lida — não some enquanto aberta)
      // - manuallyUnread (marcadas via right-click → Marcar como não lida)
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
      // Ao trocar de conversa (ou fechar), libera a sticky-unread anterior
      // para que ela saia da aba "Não lidas". Se reentrar na mesma conversa,
      // o id continua sendo o mesmo e mantemos.
      if (this.stickyUnreadId && conv?.id !== this.stickyUnreadId) {
        this.stickyUnreadId = null
      }
    },
    // Marca uma conversa como lida (estado local). Se ela tinha unread > 0,
    // pina o id em `stickyUnreadId` para que continue visível na aba
    // "Não lidas" enquanto estiver aberta. A chamada HTTP para persistir o
    // last_seen é feita em ConversationsThread (best-effort).
    markRead(id: string) {
      // Abrir a conversa também limpa qualquer "marcada como não lida" manual,
      // espelhando o WhatsApp Web — reentrar na conversa = lida de novo. Se
      // havia marcação manual ou unread real, pina a sticky pra a conversa
      // continuar visível na aba "Não lidas" enquanto estiver aberta.
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
        // Decremento otimista do badge "Não lidas" — reconciliado pelo
        // próximo loadMeta() (debounced). Sem isso, há flicker de ~200ms
        // entre a leitura local e o refetch do backend.
        this.bumpMetaUnread(conv.status, -1)
      }
    },
    // Aplica delta otimista no contador de unread do meta (clamp em 0). Usado
    // por markRead para decrementar imediatamente. Reconciliação fica por
    // conta do próximo loadMeta() (chamado via invalidateMeta debounced ou
    // ao trocar de filtro).
    bumpMetaUnread(status: ConversationStatus, delta: number) {
      const map: Record<ConversationStatus, keyof ConversationMeta> = {
        0: 'open', 1: 'resolved', 2: 'pending', 3: 'snoozed'
      }
      const k = map[status]
      const bucket = this.meta[k]
      if (!bucket) return
      bucket.unread = Math.max(0, bucket.unread + delta)
    },
    // Marca uma conversa como não lida manualmente — igual ao right-click do
    // WhatsApp Web. Local-only: não muda assignee_last_seen_at, não afeta
    // read receipts. A conversa volta a aparecer no filtro "Não lidas" e
    // ganha o dot indicador na lista. Solta a sticky se for a mesma id pra
    // a "marca manual" prevalecer mesmo se o agente reabrir o filtro.
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
      // `current` is a separate reference from `list[idx]`. Without this,
      // ConversationsIndex's `selected` computed (which reads `current` first)
      // returns the stale object after upsert, so child props don't update.
      if (this.current?.id === conv.id) this.current = conv

      // Warm message bucket if empty. Messages are owned by useMessagesStore
      // and must not be touched or overwritten here.
      useMessagesStore().warmIfEmpty(conv)
    },
    // applyPatch só mescla campos numa conversa já carregada (em list e/ou
    // current). Diferente de `upsert`, NÃO insere uma conversa nova — usado
    // por handlers realtime que querem refletir mudanças sem injetar a
    // conversa em filtros que não a contêm (ex.: usuário filtrando "Resolvidas"
    // recebe message.created da conversa aberta `current` que está Open).
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
      // Trocar tab/status/inbox/label/etc é uma ação intencional do agente —
      // a sticky-unread deixa de ser desejável (a conversa já foi lida e o
      // filtro está sendo reaplicado). Solta o pin pra que a conversa saia
      // da lista de não lidas se já não tem mais unreadCount.
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
