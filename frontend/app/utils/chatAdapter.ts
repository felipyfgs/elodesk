import { format } from 'date-fns'
import type { Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'

// --- Contact helpers (S3.1) ---

export function resolveContactName(c: Conversation): string {
  return c.meta?.sender?.name
    ?? c.meta?.sender?.phoneNumber
    ?? c.meta?.sender?.email
    ?? ''
}

export function resolveContactIdentifier(c: Conversation): string {
  return c.meta?.sender?.phoneNumber
    ?? c.meta?.sender?.email
    ?? c.meta?.sender?.identifier
    ?? ''
}

export function resolveContactAvatar(c: Conversation): string | undefined {
  return c.meta?.sender?.thumbnail
    ?? c.meta?.sender?.avatarUrl
    ?? undefined
}

// --- Message role mapping (S1.3) ---

export type ChatRole = 'user' | 'assistant' | 'system'

export function messageRole(m: Message): ChatRole {
  if (m.messageType === 2 || m.messageType === 3) return 'system'
  if (m.messageType === 1) return 'user'
  return 'assistant'
}

export function messageVariant(m: Message): 'solid' | 'outline' | 'soft' | 'subtle' | 'naked' {
  if (m.messageType === 2 || m.messageType === 3) return 'naked'
  return 'solid'
}

export function messageSide(m: Message): 'left' | 'right' {
  return m.messageType === 1 ? 'right' : 'left'
}

// --- Bubble kind (S2.5) ---

export type BubbleKind = 'text' | 'attachment' | 'private' | 'deleted' | 'activity' | 'template' | 'error' | 'empty'

function messageContentAttributes(m: Message): Record<string, unknown> {
  const attrs = m.contentAttributes
  if (!attrs) return {}
  if (typeof attrs === 'string') {
    try {
      return JSON.parse(attrs) as Record<string, unknown>
    } catch {
      return {}
    }
  }
  return attrs
}

export function messageBubbleKind(m: Message): BubbleKind {
  const ca = messageContentAttributes(m)
  if (ca?.deleted) return 'deleted'
  if (m.private || ca?.private) return 'private'
  if (m.messageType === 2) return 'activity'
  if (m.messageType === 3) return 'template'
  if (m.status === 3) return 'error' // MessageStatus.Failed
  if (!m.content && !hasAttachments(m)) return 'empty'
  if (hasAttachments(m) && !m.content) return 'attachment'
  return 'text'
}

// --- Attachment helpers (S1.2) ---

export interface MessageAttachment {
  fileUrl?: string
  path?: string
  fileType: string
  size?: number
}

export function hasAttachments(m: Message): boolean {
  if (m.attachments && m.attachments.length > 0) return true
  const ca = messageContentAttributes(m)
  const attachments = ca?.attachments as MessageAttachment[] | undefined
  return !!(attachments && attachments.length > 0)
}

// Backend AttachmentFileType enum: 0=image, 1=audio, 2=video, 3=file,
// 4=location, 5=fallback. When the server-side contentType (MIME) is
// absent we fall back to mapping the numeric enum to a string token the
// UI can pattern-match against.
const FILE_TYPE_MAP: Record<number, string> = {
  0: 'image',
  1: 'audio',
  2: 'video',
  3: 'file',
  4: 'location',
  5: 'file'
}

function normalizeFileType(contentType: string | undefined | null, rawFileType: unknown): string {
  if (contentType) return contentType
  if (typeof rawFileType === 'string') return rawFileType
  if (typeof rawFileType === 'number') return FILE_TYPE_MAP[rawFileType] ?? 'file'
  return 'file'
}

export function getAttachments(m: Message): MessageAttachment[] {
  if (m.attachments && m.attachments.length > 0) {
    return m.attachments.map(a => ({
      path: a.fileKey,
      fileType: normalizeFileType(a.contentType, a.fileType),
      size: a.size
    }))
  }
  const ca = messageContentAttributes(m)
  return (ca?.attachments as MessageAttachment[] | undefined) ?? []
}

// --- Status mapping (S1.4) ---

export interface StatusDisplay {
  icon: string
  label: string
  color: string
}

export function messageStatusDisplay(m: Message, t: (key: string) => string): StatusDisplay {
  switch (m.status) {
    case 2: return { icon: 'i-lucide-check-check', label: t('conversations.message.status.READ'), color: 'text-primary' }
    case 1: return { icon: 'i-lucide-check-check', label: t('conversations.message.status.DELIVERED'), color: 'text-muted' }
    case 0: return { icon: 'i-lucide-check', label: t('conversations.message.status.SENT'), color: 'text-muted' }
    case 3: return { icon: 'i-lucide-alert-circle', label: t('conversations.message.status.FAILED'), color: 'text-error' }
    default: return { icon: 'i-lucide-clock', label: '', color: 'text-muted' }
  }
}

// --- Message grouping (S2.9) ---

function senderKey(m: Message): string {
  // Prefer the hydrated `sender` struct (new Chatwoot shape); fall back to the
  // legacy senderType/senderId pair used by optimistic composer messages.
  if (m.sender?.id != null) return `${m.sender.type ?? ''}:${m.sender.id}`
  return `${m.senderType ?? ''}:${m.senderId ?? ''}`
}

export function shouldGroupWith(prev: Message, curr: Message): boolean {
  if (senderKey(prev) !== senderKey(curr)) return false
  if (prev.messageType !== curr.messageType) return false
  if (curr.status === 3 || prev.status === 3) return false // failed
  const prevTime = new Date(prev.createdAt).getTime()
  const currTime = new Date(curr.createdAt).getTime()
  return Math.abs(currTime - prevTime) < 60_000 // same minute
}

// --- Message parts (S2.3) ---

export function messageParts(m: Message) {
  const parts: { type: 'text', text: string }[] = []
  if (m.content) {
    parts.push({ type: 'text', text: m.content })
  }
  return parts
}

// --- Time formatting ---

export function messageTime(m: Message): string {
  if (!m.createdAt) return ''
  const d = new Date(m.createdAt)
  if (Number.isNaN(d.getTime())) return ''
  return format(d, 'HH:mm')
}
