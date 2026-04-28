import { useWebSocket } from '@vueuse/core'
import type { Ref, ShallowRef } from 'vue'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'
import { refreshAccessToken } from '~/composables/useApi'
import { normalizeApiResponse } from '~/utils/apiAdapter'

// Backend closes WS with 1008 (PolicyViolation) when the JWT in the URL is
// invalid/expired. autoReconnect would loop on the same stale token, so we
// intercept and refresh first.
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

// Module-scoped singleton: one WebSocket per browser tab, period.
// useWebSocket() called multiple times opens N connections — that was the
// previous bug (joinAccount/joinConversation/sendRaw each spawned a new WS),
// so events were silently dropped on whatever the most recent connection
// happened to be.
let socketInstance: SocketInstance | null = null

// Refresh proativo: o backend rejeita o WS com close 1008 quando o JWT
// expira (JWT_ACCESS_TTL no servidor; default 15m). Reagir só no close deixa
// um buraco entre o evento de auth-fail e a reconexão — qualquer
// message.created que chega nesse intervalo é perdido. Por isso decodificamos
// o `exp` do token e disparamos `refreshAccessToken` 60s antes da expiração,
// fechando o socket atual e abrindo um novo com o token recém-emitido.
let tokenRefreshTimer: ReturnType<typeof setTimeout> | null = null

function decodeJwtExp(token: string): number | null {
  const parts = token.split('.')
  if (parts.length !== 3) return null
  const payload = parts[1]
  if (!payload) return null
  try {
    // JWT usa base64url; converter pra base64 padrão antes de atob.
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
  // Token já expirou ou está a <60s de expirar: dispara imediato.
  const delay = Math.max(0, refreshInMs)
  tokenRefreshTimer = setTimeout(() => {
    tokenRefreshTimer = null
    const auth = useAuthStore()
    if (!auth.refreshToken) return
    refreshAccessToken()
      .then(() => {
        if (auth.accessToken) getOrCreateSocket(auth.accessToken)
      })
      .catch(() => { /* useApi 401 path will redirect to /login */ })
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
        const state = useRealtimeState()
        state.handlers.clear()
        state.joined.accounts.clear()
        state.joined.inboxes.clear()
        state.joined.conversations.clear()
        socketInstance = null
        if (tokenRefreshTimer) {
          clearTimeout(tokenRefreshTimer)
          tokenRefreshTimer = null
        }
      }
    },
    onConnected() {
      // After the socket opens (initial or after reconnect) replay every
      // join we accumulated, so events keep flowing without the caller
      // having to retry.
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
        .catch(() => { /* useApi 401 path will redirect to /login */ })
    },
    onMessage(_ws, event) {
      try {
        const msg = JSON.parse(event.data as string)
        const state = useRealtimeState()
        if (msg.type && state.handlers.has(msg.type)) {
          // Backend broadcasts snake_case + epoch-second timestamps (same shape
          // as REST). REST goes through apiAdapter in useApi; the WebSocket
          // path bypassed it, so handlers received raw snake_case payloads
          // (e.g. msg.conversationId === undefined → wrong bucket key).
          const payload = normalizeApiResponse<Record<string, unknown>>(msg.payload)
          for (const handler of state.handlers.get(msg.type)!) {
            handler(payload)
          }
        }
      } catch { /* ignore non-JSON */ }
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
  // The room set already records the intent; if the socket isn't OPEN yet,
  // onConnected → rejoinAllRooms will flush this once it opens.
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
  }

  return { connect: ensureConnected, disconnect, joinAccount, joinInbox, joinConversation, leaveConversation, on }
}
