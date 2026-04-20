import { useWebSocket } from '@vueuse/core'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'

interface JoinState {
  accounts: Set<string>
  inboxes: Set<string>
  conversations: Set<string>
}

const joined: JoinState = {
  accounts: new Set(),
  inboxes: new Set(),
  conversations: new Set()
}

type MessageHandler = (payload: Record<string, unknown>) => void
const handlers = new Map<string, Set<MessageHandler>>()
let storeHandlersInitialized = false

function sendRaw(token: string, data: Record<string, unknown>) {
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
        handlers.clear()
        joined.accounts.clear()
        joined.inboxes.clear()
        joined.conversations.clear()
      }
    },
    onMessage(_ws, event) {
      try {
        const msg = JSON.parse(event.data as string)
        if (msg.type && handlers.has(msg.type)) {
          for (const handler of handlers.get(msg.type)!) {
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
      for (const id of joined.accounts) sendRaw(token, { type: 'join.account', payload: { id: Number(id) } })
      for (const id of joined.inboxes) sendRaw(token, { type: 'join.inbox', payload: { id: Number(id) } })
      for (const id of joined.conversations) sendRaw(token, { type: 'join.conversation', payload: { id: Number(id) } })
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
    joined.accounts.add(accountId)
    if (!auth.accessToken) return
    ensureConnected()
    const { status } = getOrCreateSocket(auth.accessToken)
    if (status.value === 'OPEN') {
      sendRaw(auth.accessToken, { type: 'join.account', payload: { id: Number(accountId) } })
    }
  }

  function joinInbox(inboxId: string) {
    joined.inboxes.add(inboxId)
    if (!auth.accessToken) return
    ensureConnected()
    const { status } = getOrCreateSocket(auth.accessToken)
    if (status.value === 'OPEN') {
      sendRaw(auth.accessToken, { type: 'join.inbox', payload: { id: Number(inboxId) } })
    }
  }

  function joinConversation(conversationId: string) {
    joined.conversations.add(conversationId)
    if (!auth.accessToken) return
    ensureConnected()
    const { status } = getOrCreateSocket(auth.accessToken)
    if (status.value === 'OPEN') {
      sendRaw(auth.accessToken, { type: 'join.conversation', payload: { id: Number(conversationId) } })
    }
  }

  function leaveConversation(conversationId: string) {
    joined.conversations.delete(conversationId)
  }

  function on<T = unknown>(event: string, handler: (payload: T) => void): () => void {
    if (!handlers.has(event)) handlers.set(event, new Set())
    handlers.get(event)!.add(handler as MessageHandler)
    return () => {
      handlers.get(event)?.delete(handler as MessageHandler)
    }
  }

  function disconnect() {
    if (auth.accessToken) {
      const { close } = getOrCreateSocket(auth.accessToken)
      close()
    }
    handlers.clear()
    joined.accounts.clear()
    joined.inboxes.clear()
    joined.conversations.clear()
    storeHandlersInitialized = false
  }

  if (!storeHandlersInitialized) {
    storeHandlersInitialized = true
    const labelsStore = useLabelsStore()
    on('label.deleted', (payload: Record<string, unknown>) => {
      labelsStore.remove(String(payload.label_id))
    })
  }

  return { connect: ensureConnected, disconnect, rejoinAll, joinAccount, joinInbox, joinConversation, leaveConversation, on }
}
