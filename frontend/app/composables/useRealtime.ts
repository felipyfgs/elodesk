import { useWebSocket } from '@vueuse/core'
import type { Ref, ShallowRef } from 'vue'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'
import { usePipelinesStore } from '~/stores/pipelines'
import { usePipelineCardsStore } from '~/stores/pipelineCards'
import { refreshAccessToken } from '~/composables/useApi'
import { normalizeApiResponse } from '~/utils/apiAdapter'

const WS_CLOSE_AUTH_FAILED = 1008

interface JoinState {
  accounts: Set<string>
  inboxes: Set<string>
  conversations: Set<string>
}

type MessageHandler = (payload: Record<string, unknown>) => void

interface RealtimeState {
  joined: JoinState
  handlers: Map<string, Set<MessageHandler>>
  storeHandlersInitialized: boolean
}

interface SocketInstance {
  status: Ref<'OPEN' | 'CONNECTING' | 'CLOSED'>
  ws: ShallowRef<WebSocket | undefined>
  open: () => void
  close: () => void
  token: string
}

let socketInstance: SocketInstance | null = null

let tokenRefreshTimer: ReturnType<typeof setTimeout> | null = null

function decodeJwtExp(token: string): number | null {
  const parts = token.split('.')
  if (parts.length !== 3) return null
  const payload = parts[1]
  if (!payload) return null
  try {
    const b64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const pad = (4 - (b64.length % 4)) % 4
    const decoded = atob(b64 + '='.repeat(pad))
    const obj = JSON.parse(decoded) as { exp?: unknown }
    return typeof obj.exp === 'number' ? obj.exp : null
  } catch {
    return null
  }
}

function scheduleProactiveRefresh(token: string) {
  if (tokenRefreshTimer) {
    clearTimeout(tokenRefreshTimer)
    tokenRefreshTimer = null
  }
  const exp = decodeJwtExp(token)
  if (!exp) return
  const refreshInMs = exp * 1000 - Date.now() - 60_000
  const delay = Math.max(0, refreshInMs)
  tokenRefreshTimer = setTimeout(() => {
    tokenRefreshTimer = null
    const auth = useAuthStore()
    if (!auth.refreshToken) return
    refreshAccessToken()
      .then(() => {
        if (auth.accessToken) getOrCreateSocket(auth.accessToken)
      })
      .catch(() => { })
  }, delay)
}

function useRealtimeState(): RealtimeState {
  const state = useState<RealtimeState>('realtime-state', () => ({
    joined: { accounts: new Set(), inboxes: new Set(), conversations: new Set() },
    handlers: new Map(),
    storeHandlersInitialized: false
  }))
  return state.value
}

function rejoinAllRooms(inst: SocketInstance, joined: JoinState) {
  if (inst.status.value !== 'OPEN' || !inst.ws.value) return
  for (const id of joined.accounts) {
    inst.ws.value.send(JSON.stringify({ type: 'join.account', payload: { id: Number(id) } }))
  }
  for (const id of joined.inboxes) {
    inst.ws.value.send(JSON.stringify({ type: 'join.inbox', payload: { id: Number(id) } }))
  }
  for (const id of joined.conversations) {
    inst.ws.value.send(JSON.stringify({ type: 'join.conversation', payload: { id: Number(id) } }))
  }
}

function getOrCreateSocket(token: string): SocketInstance {
  if (socketInstance && socketInstance.token === token) {
    return socketInstance
  }
  if (socketInstance) {
    socketInstance.close()
    socketInstance = null
  }

  const runtime = useRuntimeConfig()
  const base = runtime.public.wsUrl.replace(/^http/, 'ws')
  const sep = base.includes('?') ? '&' : '?'
  const url = `${base}/realtime${sep}token=${encodeURIComponent(token)}`

  const { status, ws, open, close } = useWebSocket(url, {
    heartbeat: { message: '{"type":"ping"}', interval: 30_000, pongTimeout: 10_000 },
    autoReconnect: {
      retries: 10,
      delay: 1000,
      onFailed() {
        socketInstance = null
        if (tokenRefreshTimer) {
          clearTimeout(tokenRefreshTimer)
          tokenRefreshTimer = null
        }
      }
    },
    onConnected() {
      if (socketInstance) {
        rejoinAllRooms(socketInstance, useRealtimeState().joined)
      }
    },
    onDisconnected(_ws, ev) {
      if (ev.code !== WS_CLOSE_AUTH_FAILED) return
      const closing = socketInstance
      if (closing) {
        closing.close()
        socketInstance = null
      }
      const auth = useAuthStore()
      if (!auth.refreshToken) return
      refreshAccessToken()
        .then(() => {
          if (auth.accessToken) getOrCreateSocket(auth.accessToken)
        })
        .catch(() => { })
    },
    onMessage(_ws, event) {
      try {
        const msg = JSON.parse(event.data as string)
        const state = useRealtimeState()
        if (msg.type && state.handlers.has(msg.type)) {
          const payload = normalizeApiResponse<Record<string, unknown>>(msg.payload)
          for (const handler of state.handlers.get(msg.type)!) {
            handler(payload)
          }
        }
      } catch { void 0 }
    }
  })

  socketInstance = {
    status: status as Ref<'OPEN' | 'CONNECTING' | 'CLOSED'>,
    ws,
    open,
    close,
    token
  }
  scheduleProactiveRefresh(token)
  return socketInstance
}

function sendOrQueue(inst: SocketInstance, data: Record<string, unknown>) {
  if (inst.status.value === 'OPEN' && inst.ws.value) {
    inst.ws.value.send(JSON.stringify(data))
  }
}

export const useRealtime = () => {
  const auth = useAuthStore()
  const state = useRealtimeState()

  function ensureConnected(): SocketInstance | null {
    if (!auth.accessToken) return null
    return getOrCreateSocket(auth.accessToken)
  }

  function joinAccount(accountId: string) {
    state.joined.accounts.add(accountId)
    const inst = ensureConnected()
    if (inst) sendOrQueue(inst, { type: 'join.account', payload: { id: Number(accountId) } })
  }

  function joinInbox(inboxId: string) {
    state.joined.inboxes.add(inboxId)
    const inst = ensureConnected()
    if (inst) sendOrQueue(inst, { type: 'join.inbox', payload: { id: Number(inboxId) } })
  }

  function joinConversation(conversationId: string) {
    state.joined.conversations.add(conversationId)
    const inst = ensureConnected()
    if (inst) sendOrQueue(inst, { type: 'join.conversation', payload: { id: Number(conversationId) } })
  }

  function leaveConversation(conversationId: string) {
    state.joined.conversations.delete(conversationId)
    const inst = ensureConnected()
    if (inst) sendOrQueue(inst, { type: 'leave.conversation', payload: { id: Number(conversationId) } })
  }

  function on<T = unknown>(event: string, handler: (payload: T) => void): () => void {
    if (!state.handlers.has(event)) state.handlers.set(event, new Set())
    state.handlers.get(event)!.add(handler as MessageHandler)
    return () => {
      state.handlers.get(event)?.delete(handler as MessageHandler)
    }
  }

  function disconnect() {
    if (socketInstance) {
      socketInstance.close()
      socketInstance = null
    }
    state.handlers.clear()
    state.joined.accounts.clear()
    state.joined.inboxes.clear()
    state.joined.conversations.clear()
    state.storeHandlersInitialized = false
  }

  if (!state.storeHandlersInitialized) {
    state.storeHandlersInitialized = true
    const labelsStore = useLabelsStore()
    on('label.deleted', (payload: Record<string, unknown>) => {
      labelsStore.remove(String(payload.labelId))
    })

    const pipelinesStore = usePipelinesStore()
    const cardsStore = usePipelineCardsStore()

    on('pipeline.created', (payload: Record<string, unknown>) => {
      const id = payload.pipelineId
      if (id !== undefined && id !== null) pipelinesStore.fetchOne(id as number | string).catch(() => { })
    })
    on('pipeline.updated', (payload: Record<string, unknown>) => {
      const id = payload.pipelineId
      if (id !== undefined && id !== null) pipelinesStore.fetchOne(id as number | string).catch(() => { })
    })
    on('pipeline.archived', (payload: Record<string, unknown>) => {
      const id = payload.pipelineId
      if (id === undefined || id === null) return
      const existing = pipelinesStore.byId(id as number | string)
      if (existing) {
        pipelinesStore.upsert({ ...existing, archivedAt: new Date().toISOString() })
      }
    })

    on('stage.created', (payload: Record<string, unknown>) => {
      const id = payload.pipelineId
      if (id !== undefined && id !== null) pipelinesStore.fetchOne(id as number | string).catch(() => { })
    })
    on('stage.updated', (payload: Record<string, unknown>) => {
      const id = payload.pipelineId
      if (id !== undefined && id !== null) pipelinesStore.fetchOne(id as number | string).catch(() => { })
    })
    on('stage.deleted', (payload: Record<string, unknown>) => {
      const pipelineId = payload.pipelineId
      const stageId = payload.stageId
      if (pipelineId === undefined || pipelineId === null) return
      if (stageId === undefined || stageId === null) return
      pipelinesStore.removeStage(pipelineId as number | string, stageId as number | string)
    })

    on('card.created', (payload: Record<string, unknown>) => {
      const cardId = payload.cardId
      if (cardId !== undefined && cardId !== null) cardsStore.fetchCard(cardId as number | string).catch(() => { })
    })
    on('card.updated', (payload: Record<string, unknown>) => {
      const cardId = payload.cardId
      if (cardId !== undefined && cardId !== null) cardsStore.fetchCard(cardId as number | string).catch(() => { })
    })
    on('card.moved', (payload: Record<string, unknown>) => {
      const cardId = payload.cardId
      const pipelineId = payload.pipelineId
      const fromStageId = payload.fromStageId
      const toStageId = payload.toStageId
      const position = payload.position
      if (
        cardId === undefined || cardId === null
        || pipelineId === undefined || pipelineId === null
        || fromStageId === undefined || fromStageId === null
        || toStageId === undefined || toStageId === null
        || typeof position !== 'number'
      ) return
      cardsStore.applyRealtimeMove({
        cardId: cardId as number | string,
        pipelineId: pipelineId as number | string,
        fromStageId: fromStageId as number | string,
        toStageId: toStageId as number | string,
        position
      })
    })
    on('card.deleted', (payload: Record<string, unknown>) => {
      const pipelineId = payload.pipelineId
      const cardId = payload.cardId
      if (pipelineId === undefined || pipelineId === null) return
      if (cardId === undefined || cardId === null) return
      cardsStore.remove(pipelineId as number | string, cardId as number | string)
    })
  }

  return { connect: ensureConnected, disconnect, joinAccount, joinInbox, joinConversation, leaveConversation, on }
}
