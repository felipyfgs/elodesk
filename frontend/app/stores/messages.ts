import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

// Backend sends numeric enums; use these helpers/consts for UI mapping.
// MessageType: 0=Incoming, 1=Outgoing, 2=Activity, 3=Template
export type MessageType = 0 | 1 | 2 | 3
// MessageStatus: 0=Sent, 1=Delivered, 2=Read, 3=Failed | 'sending' (optimistic)
export type MessageStatus = 0 | 1 | 2 | 3 | 'sending'

export interface MessageAttachmentResp {
  id: number
  messageId: number
  // Backend sends AttachmentFileType as numeric enum; chatAdapter normalizes.
  fileType: string | number
  fileKey?: string
  // Nome original do arquivo (com acentos/espaços/parênteses) preservado
  // pelo backend separado da chave do MinIO (que é sanitizada).
  fileName?: string
  // URL externa (CDN do Meta/Telegram) usada quando o backend ainda não
  // baixou o blob pro MinIO. Frontend prefere fileKey (signed URL); cai pra
  // externalUrl quando fileKey não existe.
  externalUrl?: string
  extension?: string
  contentType?: string
  size?: number
  // ISO string (optimistic/realtime) or epoch ms (apiAdapter-normalized REST).
  createdAt: string | number
}

// MessageSender mirrors backend dto.MessageSenderResp — polymorphic sender
// embedded in MessageResp. `type` matches Chatwoot's lowercase tokens
// (contact/user/agent_bot); the legacy uppercase `senderType` field below is
// kept for in-memory optimistic messages built by the composer.
export interface MessageSender {
  id?: string
  name?: string
  type?: 'contact' | 'user' | 'agent_bot'
  thumbnail?: string | null
  avatarUrl?: string | null
}

export interface Message {
  id: string
  conversationId: string
  inboxId: string
  accountId: string
  content: string | null
  contentType: string
  messageType: MessageType
  // Backend now embeds the polymorphic sender as a struct; senderType/senderId
  // are kept optional for legacy callers and optimistic placeholders.
  sender?: MessageSender | null
  senderType?: 'CONTACT' | 'USER' | 'SYSTEM'
  senderId?: string | null
  sourceId: string | null
  echoId?: string | null
  private?: boolean
  status: MessageStatus
  contentAttributes: Record<string, unknown> | string | null
  forwardedFromMessageId?: number | null
  attachments?: MessageAttachmentResp[]
  // ISO string (optimistic/realtime) or epoch ms (apiAdapter-normalized REST).
  createdAt: string | number
  updatedAt: string | number
}

// Forward types
export type ForwardTarget
  = | { conversationId: string }
    | { contactId: string, inboxId: string }

export interface ForwardResult {
  target: ForwardTarget
  status: 'success' | 'failed'
  createdMessageIds?: number[]
  conversationId?: number
  createdConversation?: boolean
  error?: string
}

export interface ForwardMessagesResp {
  results: ForwardResult[]
}

export const useMessagesStore = defineStore('messages', {
  state: () => ({
    byConversation: {} as Record<string, Message[]>,
    // Per-conversation reply draft. The composer reads from this to show a
    // quoted preview and embeds the in_reply_to payload on send.
    replyingTo: {} as Record<string, Message | null>,
    // Dedup de fetches concorrentes: hover-prefetch + click podem chamar o
    // endpoint quase simultaneamente. Set < Map por simplicidade — só
    // precisamos saber "tem request em voo agora?".
    inflight: new Set<string>(),
    // Timestamp do último fetch bem-sucedido por conversa. Usado pelo
    // prefetch como TTL anti-hammer (não chama de novo se acabou de cachear).
    fetchedAt: {} as Record<string, number>
  }),
  actions: {
    set(conversationId: string, list: Message[]) {
      this.byConversation[conversationId] = list
    },
    // fetchMessages é o caminho único pra puxar mensagens de uma conversa —
    // tanto o click (Thread.vue) quanto o hover (List.vue) passam por aqui pra
    // dedupe automático. mergeFetched preserva itens vindos por WS.
    async fetchMessages(conversationId: string, opts?: { freshMs?: number }) {
      if (!conversationId) return
      if (this.inflight.has(conversationId)) return
      const fresh = opts?.freshMs ?? 0
      if (fresh > 0) {
        const last = this.fetchedAt[conversationId] ?? 0
        if (last && Date.now() - last < fresh) return
      }
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      this.inflight.add(conversationId)
      try {
        const res = await api<{ payload: Message[] }>(
          `/accounts/${auth.account.id}/conversations/${conversationId}/messages`
        )
        if (res.payload) {
          this.mergeFetched(conversationId, [...res.payload].reverse())
        }
        this.fetchedAt[conversationId] = Date.now()
      } catch (err) {
        // Best-effort: hover-prefetch não deve incomodar o agente com toast
        // de erro; o click-fetch retenta logo em seguida.
        console.error('[messages] fetch failed', err)
      } finally {
        this.inflight.delete(conversationId)
      }
    },
    // Prefetch chamado on-hover: pula quando já tem mensagem cacheada (basta
    // mostrar) ou quando acabamos de cachear (anti-hammer em scroll rápido).
    prefetch(conversationId: string) {
      if (!conversationId) return
      if ((this.byConversation[conversationId]?.length ?? 0) > 0) return
      void this.fetchMessages(conversationId, { freshMs: 30_000 })
    },
    // mergeFetched aplica o resultado paginado do fetch REST sobre o bucket
    // existente sem perder mensagens que chegaram via realtime durante o
    // request. Estratégia:
    //   1. mantemos as mensagens já presentes (incluindo tmp: otimistas)
    //   2. para cada item do REST, fazemos upsert (substitui se id bate;
    //      reconcilia echoId; insere ordenado caso contrário)
    // Antes, `set(...)` substituía o bucket e descartava qualquer mensagem
    // recém-chegada por WS — race clássica abrir conversa + nova mensagem.
    mergeFetched(conversationId: string, list: Message[]) {
      if (!this.byConversation[conversationId]) {
        // Bucket vazio: caminho rápido — atribuição direta sem ordenação extra
        // (REST já vem ordenado por created_at desc; o caller inverte).
        this.byConversation[conversationId] = list
        return
      }
      for (const m of list) this.upsert(m)
    },
    setReplyTarget(conversationId: string, msg: Message | null) {
      this.replyingTo[conversationId] = msg
    },
    clearReplyTarget(conversationId: string) {
      this.replyingTo[conversationId] = null
    },
    upsert(msg: Message) {
      const bucket = (this.byConversation[String(msg.conversationId)] ||= [])

      // Reconcile optimistic tmp messages by echoId: when the real server
      // message arrives it replaces the `tmp:<echoId>` placeholder.
      if (msg.echoId) {
        const tmpTarget = `tmp:${msg.echoId}`
        const tmpIdx = bucket.findIndex((m) => {
          const id = String(m.id)
          return id === tmpTarget || (id.startsWith('tmp:') && m.echoId === msg.echoId)
        })
        if (tmpIdx >= 0) {
          bucket.splice(tmpIdx, 1)
        }
      }

      // createdAt pode chegar como string ISO (optimistic/realtime) ou epoch ms
      // (REST normalizado pelo apiAdapter). Date(...) aceita os dois — calcula
      // uma vez aqui pra evitar repetir no caminho de inserção.
      const ts = new Date(msg.createdAt).getTime()

      const idx = bucket.findIndex(m => String(m.id) === String(msg.id))
      if (idx >= 0) {
        // Update in-place. Se o timestamp mudou e a mensagem ficou fora de
        // ordem (raro, ex.: server backfill com timestamp corrigido) deixa o
        // sort do bloco else cuidar — aqui mantemos posição porque mensagens
        // editadas/atualizadas conservam o created_at original.
        bucket[idx] = msg
        return
      }

      // Mensagens normalmente chegam em ordem cronológica (REST inicial vem
      // ordenado, realtime entrega na ordem de criação). Inserção O(n) por
      // varredura reversa em vez de O(n log n) sort do array inteiro a cada
      // upsert — em conversas grandes (>500 mensagens) o sort era visível como
      // stutter na UI.
      let i = bucket.length - 1
      while (i >= 0 && new Date(bucket[i]!.createdAt).getTime() > ts) i--
      bucket.splice(i + 1, 0, msg)
    },
    remove(id: string | number) {
      const target = String(id)
      for (const convId of Object.keys(this.byConversation)) {
        this.byConversation[convId] = this.byConversation[convId]!.filter(m => String(m.id) !== target)
      }
    },
    async forward({ sourceMessageIds, targets }: { sourceMessageIds: string[], targets: ForwardTarget[] }) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('no account')
      const dtoTargets = targets.map((t) => {
        if ('conversationId' in t) {
          return { conversation_id: Number(t.conversationId) }
        }
        return { contact_id: Number(t.contactId), inbox_id: Number(t.inboxId) }
      })
      // useApi unwraps the { success, data } envelope, so we get
      // ForwardMessagesResp directly.
      return await api<ForwardMessagesResp>(`/accounts/${auth.account.id}/messages/forward`, {
        method: 'POST',
        body: {
          source_message_ids: sourceMessageIds.map(Number),
          targets: dtoTargets
        }
      })
    }
  }
})
