import { useWebSocket } from '@vueuse/core'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'

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

function useRealtimeState(): RealtimeState {
  // useState is request-scoped on SSR and client-scoped on the browser,
  // preventing cross-request leakage.
  const state = useState<RealtimeState>('realtime-state', () => ({
    joined: { accounts: new Set(), inboxes: new Set(), conversations: new Set() },
    handlers: new Map(),
    storeHandlersInitialized: false
  }))
  return state.value
}

function sendRaw(token: string, data: Record<string, unknown>, _joined: JoinState) {
  const { status, ws } = getOrCreateSocket(token)
  if (status.value === 'OPEN' && ws.value) {
    ws.value.send(JSON.stringify(data))
  }
}

function getOrCreateSocket(token: string) {
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
      }
    },
    onMessage(_ws, event) {
      try {
        const msg = JSON.parse(event.data as string)
        const state = useRealtimeState()
        if (msg.type && state.handlers.has(msg.type)) {
          for (const handler of state.handlers.get(msg.type)!) {
            handler(msg.payload)
          }
        }
      } catch { /* ignore non-JSON */ }
    }
  })

  return { status, ws, open, close }
}

export const useRealtime = () => {
  const auth = useAuthStore()
  const state = useRealtimeState()

  function ensureConnected() {
    if (!auth.accessToken) return
    getOrCreateSocket(auth.accessToken)
  }

  function rejoinAll() {
    if (!auth.accessToken) return
    const token = auth.accessToken
    const { status } = getOrCreateSocket(token)

    const tryRejoin = () => {
      if (status.value !== 'OPEN') return
      for (const id of state.joined.accounts) sendRaw(token, { type: 'join.account', payload: { id: Number(id) } }, state.joined)
      for (const id of state.joined.inboxes) sendRaw(token, { type: 'join.inbox', payload: { id: Number(id) } }, state.joined)
      for (const id of state.joined.conversations) sendRaw(token, { type: 'join.conversation', payload: { id: Number(id) } }, state.joined)
    }

    if (status.value === 'OPEN') {
      tryRejoin()
    } else {
      const interval = setInterval(() => {
        if (status.value === 'OPEN') {
          clearInterval(interval)
          tryRejoin()
        }
      }, 500)
    }
  }

  function joinAccount(accountId: string) {
    state.joined.accounts.add(accountId)
    if (!auth.accessToken) return
    ensureConnected()
    const { status } = getOrCreateSocket(auth.accessToken)
    if (status.value === 'OPEN') {
      sendRaw(auth.accessToken, { type: 'join.account', payload: { id: Number(accountId) } }, state.joined)
    }
  }

  function joinInbox(inboxId: string) {
    state.joined.inboxes.add(inboxId)
    if (!auth.accessToken) return
    ensureConnected()
    const { status } = getOrCreateSocket(auth.accessToken)
    if (status.value === 'OPEN') {
      sendRaw(auth.accessToken, { type: 'join.inbox', payload: { id: Number(inboxId) } }, state.joined)
    }
  }

  function joinConversation(conversationId: string) {
    state.joined.conversations.add(conversationId)
    if (!auth.accessToken) return
    ensureConnected()
    const { status } = getOrCreateSocket(auth.accessToken)
    if (status.value === 'OPEN') {
      sendRaw(auth.accessToken, { type: 'join.conversation', payload: { id: Number(conversationId) } }, state.joined)
    }
  }

  function leaveConversation(conversationId: string) {
    state.joined.conversations.delete(conversationId)
  }

  function on<T = unknown>(event: string, handler: (payload: T) => void): () => void {
    if (!state.handlers.has(event)) state.handlers.set(event, new Set())
    state.handlers.get(event)!.add(handler as MessageHandler)
    return () => {
      state.handlers.get(event)?.delete(handler as MessageHandler)
    }
  }

  function disconnect() {
    if (auth.accessToken) {
      const { close } = getOrCreateSocket(auth.accessToken)
      close()
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
      labelsStore.remove(String(payload.label_id))
    })
  }

  return { connect: ensureConnected, disconnect, rejoinAll, joinAccount, joinInbox, joinConversation, leaveConversation, on }
}
