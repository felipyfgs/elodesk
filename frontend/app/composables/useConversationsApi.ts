import {
  STATUS_CODE,
  type Conversation,
  type ConversationListMeta,
  type ConversationListResponse,
  type ConversationMeta,
  type ConversationStatusFilter
} from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'

// Query parameters accepted by GET /api/v1/accounts/:aid/conversations.
// Mirrors backend handler.ConversationHandler.List filter wiring.
export interface ListConversationsParams {
  page?: number
  perPage?: number
  status?: ConversationStatusFilter
  assigneeType?: 'mine' | 'unassigned' | 'all' | 'assigned'
  assigneeId?: string | number
  inboxId?: string | number
  teamId?: string | number
  labels?: string
  q?: string
  sortBy?: string
  unread?: boolean
  conversationType?: 'unattended' | 'mention' | 'participating'
}

function buildListQuery(p: ListConversationsParams): URLSearchParams {
  const qs = new URLSearchParams()
  if (p.page) qs.set('page', String(p.page))
  if (p.perPage) qs.set('per_page', String(p.perPage))
  if (p.status) qs.set('status', STATUS_CODE[p.status])
  if (p.assigneeType && p.assigneeType !== 'all') qs.set('assignee_type', p.assigneeType)
  if (p.assigneeId != null) qs.set('assignee_id', String(p.assigneeId))
  if (p.inboxId != null) qs.set('inbox_id', String(p.inboxId))
  if (p.teamId != null) qs.set('team_id', String(p.teamId))
  if (p.labels) qs.set('labels', p.labels)
  if (p.q) qs.set('q', p.q)
  if (p.sortBy) qs.set('sort_by', p.sortBy)
  if (p.unread) qs.set('unread', 'true')
  if (p.conversationType) qs.set('conversation_type', p.conversationType)
  return qs
}

// useConversationsApi returns a small set of typed helpers around the
// conversations REST endpoints. Centralizes the `{meta, payload}` envelope
// parsing so consumers don't have to (re)type the shape inline.
export function useConversationsApi() {
  const api = useApi()
  const auth = useAuthStore()

  function requireAccountID(): string {
    const id = auth.account?.id
    if (!id) throw new Error('useConversationsApi: no active account')
    return id
  }

  async function list(params: ListConversationsParams = {}): Promise<ConversationListResponse> {
    const accountId = requireAccountID()
    const qs = buildListQuery(params).toString()
    const url = `/accounts/${accountId}/conversations${qs ? `?${qs}` : ''}`
    return api<ConversationListResponse>(url)
  }

  async function filter(query: unknown, page = 1, perPage = 100): Promise<ConversationListResponse> {
    const accountId = requireAccountID()
    return api<ConversationListResponse>(
      `/accounts/${accountId}/conversations/filter`,
      { method: 'POST', body: { query, page, per_page: perPage } }
    )
  }

  async function show(id: string | number): Promise<Conversation> {
    const accountId = requireAccountID()
    return api<Conversation>(`/accounts/${accountId}/conversations/${id}`)
  }

  async function legacyMeta(inboxId?: string): Promise<ConversationMeta | null> {
    const accountId = requireAccountID()
    const qs = inboxId ? `?inbox_id=${encodeURIComponent(inboxId)}` : ''
    const res = await api<{ payload: ConversationMeta }>(`/accounts/${accountId}/conversations/meta${qs}`)
    return res?.payload ?? null
  }

  async function listMeta(params: Pick<ListConversationsParams, 'inboxId' | 'status'> = {}): Promise<ConversationListMeta | null> {
    // The flat `{mine_count, assigned_count, unassigned_count, all_count}` envelope
    // is delivered alongside the conversations list; this helper just calls list()
    // and discards the payload when only the totals are needed.
    const res = await list({ ...params, perPage: 1 })
    return res?.meta ?? null
  }

  return { list, filter, show, legacyMeta, listMeta }
}
