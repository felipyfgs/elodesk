<script setup lang="ts">
import type { ContextMenuItem, DropdownMenuItem } from '@nuxt/ui'
import { renderMarkdown } from '~/utils/markdown'
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore, type Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'
import {
  messageSide,
  messageBubbleKind,
  messageStatusDisplay,
  messageTime,
  hasAttachments,
  getAttachments,
  messageIsForwarded,
  messageIsForwardable,
  type BubbleKind
} from '~/utils/chatAdapter'
import { forwardSelectionModeKey, forwardSelectedIdsKey } from '~/utils/forward'

const props = defineProps<{
  message: Message
  conversation: Conversation
  grouped?: boolean
}>()

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()
const messages = useMessagesStore()

// MessageList owns the checkbox + row click; we only need the flag here to
// gate actions (context menu, chevron) and to dim non-forwardable bubbles.
// `_selectedIdsRef` is still injected so the "Encaminhar" dropdown entry can
// seed the initial selection.
const _selectionModeRef = inject(forwardSelectionModeKey, null)
const _selectedIdsRef = inject(forwardSelectedIdsKey, null)
const selectionMode = computed(() => _selectionModeRef?.value ?? false)

interface QuotedReply {
  id?: string | number
  content?: string
  author?: string
}

function messageContentAttrs(m: Message): Record<string, unknown> {
  const ca = m.contentAttributes
  if (!ca) return {}
  if (typeof ca === 'string') {
    try {
      return JSON.parse(ca) as Record<string, unknown>
    } catch {
      return {}
    }
  }
  return ca
}

const quotedReply = computed<QuotedReply | null>(() => {
  const ca = messageContentAttrs(props.message)
  const raw = ca.in_reply_to
  if (!raw || typeof raw !== 'object') return null
  const r = raw as Record<string, unknown>
  const id = typeof r.id === 'string' || typeof r.id === 'number' ? r.id : undefined
  const content = typeof r.content === 'string' ? r.content : undefined
  const author = typeof r.author === 'string' ? r.author : undefined
  return { id, content, author }
})

const bubbleKind = computed<BubbleKind>(() => messageBubbleKind(props.message))

const isMarkdown = computed(() => {
  const attrs = props.message.contentAttributes
  if (!attrs) return false
  if (typeof attrs === 'string') {
    try {
      return (JSON.parse(attrs) as { format?: string }).format === 'markdown'
    } catch {
      return false
    }
  }
  return (attrs as { format?: string }).format === 'markdown'
})

const side = computed(() => messageSide(props.message))
const isOutgoing = computed(() => side.value === 'right')
const showActions = computed(() => bubbleKind.value !== 'deleted' && bubbleKind.value !== 'activity')

// Mensagem só-com-anexo (sem texto): o balão herda largura do anexo via
// `w-fit` e usa padding enxuto, sem reservar espaço pro chevron à direita
// (chevron continua absoluto no canto, mas o PDF preview é margem suficiente).
const isAttachmentOnly = computed(() =>
  hasAttachments(props.message) && !props.message.content
)

const bubbleClass = computed(() => {
  const kind = bubbleKind.value
  let padding: string
  if (isAttachmentOnly.value) {
    padding = 'p-1.5'
  } else if (showActions.value) {
    padding = 'pl-3.5 pr-8 py-2'
  } else {
    padding = 'px-3.5 py-2'
  }
  const sizing = isAttachmentOnly.value ? 'w-fit' : ''
  if (kind === 'private') {
    return `${padding} ${sizing} text-sm shadow-sm whitespace-pre-wrap break-words rounded-lg rounded-br-sm bg-warning/10 text-highlighted ring-1 ring-warning/25`
  }
  return isOutgoing.value
    ? `${padding} ${sizing} text-sm font-medium shadow-sm whitespace-pre-wrap break-words rounded-lg rounded-br-sm bg-primary-700 text-white dark:bg-primary-800`
    : `${padding} ${sizing} text-sm shadow-sm whitespace-pre-wrap break-words rounded-lg rounded-bl-sm bg-elevated text-highlighted`
})

async function copyMessage() {
  try {
    await navigator.clipboard.writeText(props.message.content ?? '')
    toast.add({ title: t('conversations.message.copied'), color: 'success' })
  } catch {
    toast.add({ title: t('conversations.message.copyFailed'), color: 'error' })
  }
}

function replyTo() {
  messages.setReplyTarget(props.conversation.id, props.message)
}

function startForward() {
  if (!_selectionModeRef || !_selectedIdsRef) return
  _selectionModeRef.value = true
  _selectedIdsRef.value = new Set([String(props.message.id)])
}

async function deleteMessage() {
  if (!auth.account?.id) return
  if (!confirm(t('conversations.message.confirmDelete'))) return
  try {
    await api(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages/${props.message.id}`, {
      method: 'DELETE'
    })
    messages.remove(props.message.id)
  } catch (err) {
    console.error('[MessageBubble] delete failed', err)
    toast.add({ title: t('conversations.message.deleteFailed'), color: 'error' })
  }
}

function downloadFirstAttachment() {
  const att = getAttachments(props.message)[0]
  if (!att) return
  const href = att.fileUrl ?? (att.path ? `${useRuntimeConfig().public.apiUrl}/accounts/${auth.account?.id}/uploads/download?path=${encodeURIComponent(att.path)}` : null)
  if (!href) return
  const a = document.createElement('a')
  a.href = href
  a.download = att.path?.split('/').pop() ?? 'attachment'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

const messageActionItems = computed<DropdownMenuItem[][]>(() => {
  const groups: DropdownMenuItem[][] = []
  const primary: DropdownMenuItem[] = [
    {
      label: t('conversations.message.actions.reply'),
      icon: 'i-lucide-reply',
      onSelect: () => replyTo()
    }
  ]
  const nonForwardable = bubbleKind.value === 'activity' || bubbleKind.value === 'deleted'
  if (!nonForwardable) {
    primary.push({
      label: t('conversations.forward.triggerAction'),
      icon: 'i-lucide-forward',
      onSelect: () => startForward()
    })
  }
  if (props.message.content) {
    primary.push({
      label: t('conversations.message.actions.copy'),
      icon: 'i-lucide-copy',
      onSelect: () => copyMessage()
    })
  }
  if (hasAttachments(props.message)) {
    primary.push({
      label: t('conversations.message.actions.download'),
      icon: 'i-lucide-download',
      onSelect: () => downloadFirstAttachment()
    })
  }
  groups.push(primary)

  if (isOutgoing.value && bubbleKind.value !== 'deleted' && bubbleKind.value !== 'activity') {
    groups.push([{
      label: t('conversations.message.actions.delete'),
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => deleteMessage()
    }])
  }
  return groups
})

const isForwardable = computed(() => messageIsForwardable(props.message))

const menuOpen = ref(false)
function toggleMenu(open: boolean) {
  menuOpen.value = open
}
</script>

<template>
  <UContextMenu
    :items="showActions && !selectionMode ? (messageActionItems as ContextMenuItem[][]) : undefined"
    :disabled="!showActions || selectionMode"
  >
    <div
      :class="[
        'group/bubble relative',
        bubbleClass,
        selectionMode && !isForwardable ? 'opacity-50' : ''
      ]"
    >
      <!-- Chevron hidden during selection mode -->
      <UDropdownMenu
        v-if="showActions && !selectionMode"
        :items="messageActionItems"
        :open="menuOpen"
        :content="{ align: side === 'right' ? 'end' : 'start' }"
        @update:open="toggleMenu"
      >
        <button
          type="button"
          class="absolute right-1.5 top-1.5 z-10 grid size-5 place-content-center rounded-full text-current transition-opacity duration-150"
          :class="[
            menuOpen ? 'opacity-100' : 'opacity-0 group-hover/bubble:opacity-90 hover:!opacity-100'
          ]"
          :aria-label="t('conversations.message.actions.more')"
        >
          <UIcon name="i-lucide-chevron-down" class="size-4" />
        </button>
      </UDropdownMenu>

      <div
        v-if="quotedReply"
        class="mb-1.5 border-l-2 border-current/40 bg-black/10 px-2 py-1 text-[11px] leading-tight opacity-80"
      >
        <div class="font-medium">
          {{ quotedReply?.author ?? t('conversations.message.actions.reply') }}
        </div>
        <div class="line-clamp-2 whitespace-pre-wrap break-words text-xs">
          {{ quotedReply?.content || t('conversations.message.actions.attachment') }}
        </div>
      </div>

      <template v-if="bubbleKind === 'deleted'">
        <p class="font-semibold">
          {{ t('conversations.message.deleted') }}
        </p>
      </template>

      <template v-else-if="bubbleKind === 'private'">
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-if="message.content && isMarkdown" class="markdown-body" v-html="renderMarkdown(message.content ?? '')" />
        <p v-else-if="message.content">
          {{ message.content }}
        </p>
      </template>

      <template v-else-if="bubbleKind === 'activity'">
        <p>
          {{ message.content }}
        </p>
      </template>

      <template v-else-if="bubbleKind === 'error'">
        <p class="text-error">
          {{ message.content }}
        </p>
      </template>

      <template v-else-if="message.content && isMarkdown">
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="markdown-body" v-html="renderMarkdown(message.content ?? '')" />
      </template>

      <template v-else-if="message.content">
        <p class="whitespace-pre-wrap">
          {{ message.content }}
        </p>
      </template>

      <div v-if="hasAttachments(message)" class="mt-1 flex flex-col gap-2">
        <ConversationsMediaAttachment
          v-for="(att, ai) in getAttachments(message)"
          :key="ai"
          :attachment="att"
          :account-id="conversation.accountId"
          :conversation-id="conversation.id"
          :is-sticker="message.contentType === 'sticker' || (message.contentType as unknown) === 11"
        />
      </div>

      <!-- Forwarded badge -->
      <div
        v-if="messageIsForwarded(props.message)"
        class="-mb-0.5 mt-1 flex items-center gap-1 text-[10px] leading-none opacity-60"
      >
        <UIcon name="i-lucide-corner-up-right" class="size-3" />
        <span>{{ t('conversations.forward.forwardedBadge') }}</span>
      </div>

      <div
        v-if="bubbleKind !== 'activity'"
        class="-mb-0.5 mt-1 flex items-center justify-end gap-1 text-[10px] leading-none opacity-70"
      >
        <span class="tabular-nums">{{ messageTime(message) }}</span>
        <UIcon
          v-if="message.messageType === 1"
          :name="messageStatusDisplay(message, t).icon"
          :class="['size-3', messageStatusDisplay(message, t).color]"
        />
      </div>
    </div>
  </UContextMenu>
</template>

<style>
.markdown-body p { margin: 0; }
.markdown-body p + p { margin-top: 0.5rem; }
.markdown-body ul, .markdown-body ol { margin: 0.25rem 0; padding-left: 1.5rem; }
.markdown-body code { background: color-mix(in oklch, currentColor 10%, transparent); padding: 0 0.25rem; border-radius: 0.25rem; font-size: 0.85em; }
.markdown-body pre { background: color-mix(in oklch, currentColor 10%, transparent); padding: 0.5rem; border-radius: 0.375rem; overflow-x: auto; }
.markdown-body a { text-decoration: underline; }
</style>
