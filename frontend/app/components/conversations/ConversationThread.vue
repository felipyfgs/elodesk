<script setup lang="ts">
import type { ContextMenuItem, DropdownMenuItem } from '@nuxt/ui'
import { renderMarkdown } from '~/utils/markdown'
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore, type Message } from '~/stores/messages'
import { STATUS_MAP, type Conversation, type ConversationStatus } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useAgentsStore } from '~/stores/agents'
import { useTeamsStore } from '~/stores/teams'
import {
  resolveContactName,
  resolveContactIdentifier,
  resolveContactAvatar,
  messageRole,
  messageVariant,
  messageSide,
  messageBubbleKind,
  messageStatusDisplay,
  messageParts,
  messageTime,
  shouldGroupWith,
  hasAttachments,
  getAttachments,
  type BubbleKind,
  type MessageAttachment
} from '~/utils/chatAdapter'

const AUDIO_EXT_RE = /\.(mp3|wav|ogg|oga|webm|m4a|aac|opus|weba)(\?|$)/i

function isAudioAttachment(att: MessageAttachment): boolean {
  if (att.fileType?.toLowerCase().startsWith('audio')) return true
  if (att.fileUrl && AUDIO_EXT_RE.test(att.fileUrl)) return true
  if (att.path && AUDIO_EXT_RE.test(att.path)) return true
  return false
}

type StatusAction = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'

// Stores keep IDs as strings, but the backend's BodyParser expects int64 — so
// JSON `"1"` returns 400. Convert to number (or null) before sending in a body.
function toNumericId(v: string | number | null | undefined): number | null {
  if (v === null || v === undefined || v === '') return null
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) ? n : null
}

const props = withDefaults(defineProps<{
  conversation: Conversation
  showBack?: boolean
}>(), {
  showBack: false
})

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()
const messages = useMessagesStore()
const conversations = useConversationsStore()
const agents = useAgentsStore()
const teams = useTeamsStore()

const statusLoading = ref(false)
const assignmentLoading = ref(false)
const detailsOpen = ref(true)

const list = computed<Message[]>(() => messages.byConversation[props.conversation.id] ?? [])

async function loadMessages() {
  if (!auth.account?.id) return
  const res = await api<{ payload: Message[] }>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`)
  if (res.payload) {
    messages.set(props.conversation.id, [...res.payload].reverse())
  }
}

watch(() => props.conversation.id, loadMessages, { immediate: true })

// Scroll-to-bottom: handled at the outer overflow container instead of
// trusting UChatMessages :auto-scroll. Reactive list updates from the
// realtime store don't reliably trigger Nuxt UI's internal observer when
// the scroll element is the parent overflow div.
const scrollContainerRef = ref<HTMLDivElement | null>(null)
const STICK_THRESHOLD_PX = 80
const stickToBottom = ref(true)

function isNearBottom(el: HTMLElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight <= STICK_THRESHOLD_PX
}

function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
  const el = scrollContainerRef.value
  if (!el) return
  el.scrollTo({ top: el.scrollHeight, behavior })
}

function onScroll() {
  const el = scrollContainerRef.value
  if (!el) return
  // If the agent reads history (scrolls up), don't yank them back when a
  // new message arrives. Resume auto-stick once they're back near the end.
  stickToBottom.value = isNearBottom(el)
}

// Reset to bottom whenever the active conversation changes.
watch(() => props.conversation.id, () => {
  stickToBottom.value = true
  nextTick(() => scrollToBottom('auto'))
})

// React to new/updated messages. Watching length covers append; watching
// the last id covers the "pending → real" reconciliation in upsert.
watch(
  () => [list.value.length, list.value[list.value.length - 1]?.id] as const,
  () => {
    if (!stickToBottom.value) return
    nextTick(() => scrollToBottom('smooth'))
  }
)

onMounted(() => {
  if (!agents.items.length) {
    agents.fetch().catch((err) => {
      console.error('[ConversationThread] failed to fetch agents', err)
    })
  }
})

async function copyMessage(m: Message) {
  try {
    await navigator.clipboard.writeText(m.content ?? '')
    toast.add({ title: t('conversations.message.copied'), color: 'success' })
  } catch {
    toast.add({ title: t('conversations.message.copyFailed'), color: 'error' })
  }
}

function replyTo(m: Message) {
  messages.setReplyTarget(props.conversation.id, m)
}

async function deleteMessage(m: Message) {
  if (!auth.account?.id) return
  if (!confirm(t('conversations.message.confirmDelete'))) return
  try {
    await api(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages/${m.id}`, {
      method: 'DELETE'
    })
    // Realtime message.deleted will also remove it; this is just an
    // optimistic UI update for the sender.
    messages.remove(m.id)
  } catch (err) {
    console.error('[ConversationThread] delete failed', err)
    toast.add({ title: t('conversations.message.deleteFailed'), color: 'error' })
  }
}

function downloadFirstAttachment(m: Message) {
  const att = getAttachments(m)[0]
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

function quotedReply(m: Message): QuotedReply | null {
  const ca = messageContentAttrs(m)
  const raw = ca.in_reply_to
  if (!raw || typeof raw !== 'object') return null
  const r = raw as Record<string, unknown>
  const id = typeof r.id === 'string' || typeof r.id === 'number' ? r.id : undefined
  const content = typeof r.content === 'string' ? r.content : undefined
  const author = typeof r.author === 'string' ? r.author : undefined
  return { id, content, author }
}

// Chevron-button dropdown shares a single open slot so only one message
// menu is visible at a time. The right-click context menu is a separate
// UContextMenu anchored at the cursor.
const activeMenuId = ref<string | null>(null)
function isMenuOpen(id: string | number) {
  return activeMenuId.value === String(id)
}
function toggleMenu(id: string | number, open: boolean) {
  activeMenuId.value = open ? String(id) : null
}

function messageActionItems(m: Message): DropdownMenuItem[][] {
  const groups: DropdownMenuItem[][] = []
  const primary: DropdownMenuItem[] = [
    {
      label: t('conversations.message.actions.reply'),
      icon: 'i-lucide-reply',
      onSelect: () => replyTo(m)
    }
  ]
  if (m.content) {
    primary.push({
      label: t('conversations.message.actions.copy'),
      icon: 'i-lucide-copy',
      onSelect: () => copyMessage(m)
    })
  }
  if (hasAttachments(m)) {
    primary.push({
      label: t('conversations.message.actions.download'),
      icon: 'i-lucide-download',
      onSelect: () => downloadFirstAttachment(m)
    })
  }
  groups.push(primary)

  const isOutgoing = messageSide(m) === 'right'
  if (isOutgoing && bubbleKind(m) !== 'deleted' && bubbleKind(m) !== 'activity') {
    groups.push([{
      label: t('conversations.message.actions.delete'),
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => deleteMessage(m)
    }])
  }
  return groups
}

const contact = computed(() => props.conversation.meta?.sender ?? null)
const contactName = computed(() => resolveContactName(props.conversation))
const contactIdentifier = computed(() => resolveContactIdentifier(props.conversation))
const contactAvatar = computed(() => resolveContactAvatar(props.conversation))

const statusLabel = computed(() => {
  const keys = ['open', 'resolved', 'pending', 'snoozed']
  return t(`conversations.filters.${keys[props.conversation.status]}`)
})

const statusColor = computed(() => {
  switch (props.conversation.status) {
    case STATUS_MAP.OPEN: return 'success' as const
    case STATUS_MAP.PENDING: return 'warning' as const
    case STATUS_MAP.RESOLVED: return 'info' as const
    case STATUS_MAP.SNOOZED: return 'neutral' as const
    default: return 'neutral' as const
  }
})

const statusPrimaryLabel = computed(() => (
  props.conversation.status === STATUS_MAP.RESOLVED
    ? t('conversations.actions.open')
    : t('conversations.actions.resolve')
))

const statusActions: { status: StatusAction, label: string, icon: string }[] = [
  { status: 'OPEN', label: 'conversations.actions.open', icon: 'i-lucide-message-circle' },
  { status: 'PENDING', label: 'conversations.actions.pending', icon: 'i-lucide-clock' },
  { status: 'RESOLVED', label: 'conversations.actions.resolve', icon: 'i-lucide-check-circle' },
  { status: 'SNOOZED', label: 'conversations.actions.snooze', icon: 'i-lucide-bell-off' }
]

const statusItems = computed<DropdownMenuItem[][]>(() => [[
  ...statusActions.map(item => ({
    label: t(item.label),
    icon: item.icon,
    checked: props.conversation.status === STATUS_MAP[item.status],
    onSelect: () => updateStatus(item.status)
  }))
]])

const moreItems = computed<DropdownMenuItem[][]>(() => [[
  {
    label: `#${props.conversation.displayId}`,
    icon: 'i-lucide-hash',
    disabled: true
  },
  {
    label: props.conversation.inbox?.name || t('conversations.detail.noInbox'),
    icon: 'i-lucide-inbox',
    disabled: true
  }
]])

const currentAssigneeLabel = computed(() => {
  if (props.conversation.meta?.assignee?.name) return props.conversation.meta.assignee.name
  const agent = agents.items.find(a => String(a.userId) === String(props.conversation.assigneeId))
  return agent?.name ?? t('assignment.unassigned')
})

const currentTeamLabel = computed(() => {
  const team = teams.byId(props.conversation.teamId ?? '')
  return team?.name ?? t('assignment.unassigned')
})

const agentItems = computed<DropdownMenuItem[][]>(() => [
  [{
    label: t('assignment.unassigned'),
    icon: 'i-lucide-user-minus',
    checked: !props.conversation.assigneeId,
    onSelect: () => updateAssignment(null, props.conversation.teamId ?? null)
  }],
  ...agents.items.map(agent => [{
    label: agent.name || agent.email,
    checked: String(agent.userId) === String(props.conversation.assigneeId),
    onSelect: () => updateAssignment(String(agent.userId), props.conversation.teamId ?? null)
  }])
])

const teamItems = computed<DropdownMenuItem[][]>(() => [
  [{
    label: t('assignment.unassigned'),
    icon: 'i-lucide-users',
    checked: !props.conversation.teamId,
    onSelect: () => updateAssignment(props.conversation.assigneeId ?? null, null)
  }],
  ...teams.list.map(team => [{
    label: team.name,
    checked: String(team.id) === String(props.conversation.teamId),
    onSelect: () => updateAssignment(props.conversation.assigneeId ?? null, team.id)
  }])
])

const priorityItems = computed<DropdownMenuItem[][]>(() => [[
  { label: t('conversations.detail.none'), checked: true }
]])

const detailSections = computed(() => [
  { value: 'actions', label: t('conversations.detail.actions'), icon: 'i-lucide-bolt' },
  { value: 'macros', label: t('conversations.detail.macros'), icon: 'i-lucide-command' },
  { value: 'conversationInfo', label: t('conversations.detail.conversationInfo'), icon: 'i-lucide-info' },
  { value: 'contactAttributes', label: t('conversations.detail.contactAttributes'), icon: 'i-lucide-tags' },
  { value: 'contactNotes', label: t('conversations.detail.contactNotes'), icon: 'i-lucide-notebook-pen' },
  { value: 'previousConversations', label: t('conversations.detail.previousConversations'), icon: 'i-lucide-history' },
  { value: 'participants', label: t('conversations.detail.participants'), icon: 'i-lucide-users' },
  { value: 'linkedIssues', label: t('conversations.detail.linkedIssues'), icon: 'i-lucide-git-pull-request' }
])

// Mostra apenas info que NÃO está visível no header (avatar / nome /
// identifier / badges de conversa+inbox+canal). Telefone/identifier/inbox/
// canal já estão acima — exibir de novo na lista é poluição.
const contactRows = computed(() => {
  const rows: { icon: string, title: string, value: string }[] = []
  const phone = contact.value?.phoneNumber
  const identifier = contactIdentifier.value
  const email = contact.value?.email
  if (email) {
    rows.push({ icon: 'i-lucide-mail', title: t('conversations.detail.email'), value: email })
  }
  if (identifier && identifier !== phone && identifier !== email) {
    rows.push({ icon: 'i-lucide-at-sign', title: t('conversations.detail.identifier'), value: identifier })
  }
  return rows
})

const channelBadge = computed(() => {
  const type = props.conversation.inbox?.channelType || props.conversation.meta?.channel || ''
  if (type.includes('Api')) return 'API'
  if (type.includes('Whatsapp')) return 'WA'
  if (type.includes('Twilio')) return 'TW'
  if (type.includes('Sms')) return 'SMS'
  if (type.includes('Email')) return 'EM'
  if (type.includes('Instagram')) return 'IG'
  if (type.includes('Facebook')) return 'FB'
  if (type.includes('Telegram')) return 'TG'
  if (type.includes('Line')) return 'LI'
  if (type.includes('Tiktok')) return 'TK'
  if (type.includes('Twitter')) return 'TW'
  if (type.includes('WebWidget')) return 'WW'
  return type.split('::').pop() || ''
})

function unwrapConversation(res: Conversation | { payload?: Conversation } | undefined): Conversation | undefined {
  if (!res) return undefined
  if (typeof res === 'object' && 'payload' in res) return res.payload
  return res as Conversation
}

async function updateStatus(status: StatusAction) {
  const accountId = auth.account?.id
  if (!accountId || statusLoading.value) return
  statusLoading.value = true
  try {
    const res = await api<Conversation | { payload?: Conversation }>(`/accounts/${accountId}/conversations/${props.conversation.id}/status`, {
      method: 'PATCH',
      body: { status: STATUS_MAP[status] }
    })
    const conv = unwrapConversation(res)
    conversations.upsert(conv ?? { ...props.conversation, status: STATUS_MAP[status] as ConversationStatus })
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    statusLoading.value = false
  }
}

async function updateAssignment(assigneeId: string | null, teamId: string | null) {
  const accountId = auth.account?.id
  if (!accountId || assignmentLoading.value) return
  assignmentLoading.value = true
  try {
    const res = await api<Conversation | { payload?: Conversation }>(`/accounts/${accountId}/conversations/${props.conversation.id}/assignments`, {
      method: 'POST',
      body: { assignee_id: toNumericId(assigneeId), team_id: toNumericId(teamId) }
    })
    const conv = unwrapConversation(res)
    if (conv) {
      conversations.upsert(conv)
    } else {
      conversations.upsert({ ...props.conversation, assigneeId, teamId })
    }
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    assignmentLoading.value = false
  }
}

function assignToMe() {
  if (!auth.user?.id) return
  updateAssignment(auth.user.id, props.conversation.teamId ?? null)
}

function isGrouped(index: number): boolean {
  if (index === 0) return false
  const prev = list.value[index - 1]
  const curr = list.value[index]
  if (!prev || !curr) return false
  return shouldGroupWith(prev, curr)
}

function bubbleKind(m: Message): BubbleKind {
  return messageBubbleKind(m)
}

function isMarkdown(m: Message): boolean {
  const attrs = m.contentAttributes
  if (!attrs) return false
  if (typeof attrs === 'string') {
    try {
      return (JSON.parse(attrs) as { format?: string }).format === 'markdown'
    } catch {
      return false
    }
  }
  return (attrs as { format?: string }).format === 'markdown'
}

function messageUi(m: Message) {
  const kind = bubbleKind(m)
  if (kind === 'activity' || kind === 'template') {
    return {
      root: 'flex justify-center [--last-message-height:0px]',
      container: 'justify-center pb-2',
      content: '!w-fit min-h-0 rounded-lg bg-elevated px-3 py-1 text-xs text-muted',
      actions: 'hidden'
    }
  }

  if (kind === 'private') {
    return {
      root: '[--last-message-height:0px]',
      container: 'justify-end pb-4',
      content: '!p-0 !bg-transparent !ring-0 !shadow-none !rounded-none max-w-[34rem]',
      actions: 'right-1 text-warning/70'
    }
  }

  const outgoing = messageSide(m) === 'right'
  return {
    root: '[--last-message-height:0px]',
    container: outgoing ? 'justify-end pb-4' : 'justify-start pb-4',
    content: '!p-0 !bg-transparent !ring-0 !shadow-none !rounded-none max-w-[34rem]',
    actions: outgoing ? 'right-1 text-dimmed' : 'left-1 text-dimmed'
  }
}

function bubbleClass(m: Message): string {
  const kind = bubbleKind(m)
  // Reserve space on the top-right for the chevron action button so text
  // never renders underneath it. Deleted/activity bubbles skip the button
  // and can use uniform padding.
  const hasActions = kind !== 'deleted' && kind !== 'activity'
  const padding = hasActions ? 'pl-3.5 pr-8 py-2' : 'px-3.5 py-2'
  if (kind === 'private') {
    return `${padding} text-sm shadow-sm whitespace-pre-wrap break-words rounded-lg rounded-br-sm bg-warning/10 text-highlighted ring-1 ring-warning/25`
  }
  const outgoing = messageSide(m) === 'right'
  return outgoing
    ? `${padding} text-sm shadow-sm whitespace-pre-wrap break-words rounded-lg rounded-br-sm bg-primary text-inverted`
    : `${padding} text-sm shadow-sm whitespace-pre-wrap break-words rounded-lg rounded-bl-sm bg-elevated text-highlighted`
}
</script>

<template>
  <UDashboardPanel id="conversations-thread" class="min-w-0 flex-1">
    <div class="flex min-h-0 flex-1 bg-default">
      <section class="flex min-w-0 flex-1 flex-col bg-default">
        <header class="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-default px-3 sm:px-4">
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <UTooltip v-if="showBack" :text="t('common.close')">
              <UButton
                icon="i-lucide-arrow-left"
                color="neutral"
                variant="ghost"
                size="sm"
                :aria-label="t('common.close')"
                @click="emit('close')"
              />
            </UTooltip>

            <ChatHeader :conversation="conversation" class="min-w-0 flex-1" />
          </div>

          <div class="flex shrink-0 items-center gap-1">
            <UDropdownMenu :items="statusItems" :content="{ align: 'end' }">
              <UButton
                :label="statusPrimaryLabel"
                trailing-icon="i-lucide-chevron-down"
                color="neutral"
                variant="soft"
                size="sm"
                :loading="statusLoading"
              />
            </UDropdownMenu>
            <UTooltip :text="t('conversations.detail.contacts')">
              <UButton
                icon="i-lucide-user-round"
                color="neutral"
                :variant="detailsOpen ? 'soft' : 'ghost'"
                size="xs"
                class="hidden lg:inline-flex"
                :aria-label="t('conversations.detail.contacts')"
                @click="detailsOpen = !detailsOpen"
              />
            </UTooltip>
            <UDropdownMenu :items="moreItems" :content="{ align: 'end' }">
              <UButton
                icon="i-lucide-more-vertical"
                color="neutral"
                variant="ghost"
                size="xs"
                :aria-label="t('conversations.detail.more')"
              />
            </UDropdownMenu>
          </div>
        </header>

        <div class="flex min-h-0 flex-1 flex-col bg-default">
          <div
            ref="scrollContainerRef"
            class="min-h-0 flex-1 overflow-y-auto px-3 sm:px-4"
            @scroll.passive="onScroll"
          >
            <UChatMessages
              class="mx-auto w-full max-w-4xl px-0 py-4"
              :should-scroll-to-bottom="false"
              :auto-scroll="false"
              :spacing-offset="0"
            >
              <div v-if="!list.length" class="flex flex-col items-center justify-center py-12 text-muted">
                <UIcon name="i-lucide-message-circle-off" class="mb-2 size-8 text-dimmed" />
                <p class="text-sm">
                  {{ t('conversations.thread.empty') }}
                </p>
              </div>

              <UChatMessage
                v-for="(m, i) in list"
                :id="String(m.id)"
                :key="m.id"
                :role="messageRole(m)"
                :variant="messageVariant(m)"
                :side="messageSide(m)"
                :parts="messageParts(m)"
                :compact="isGrouped(i)"
                :ui="messageUi(m)"
              >
                <template #content>
                  <UContextMenu
                    :items="bubbleKind(m) === 'deleted' || bubbleKind(m) === 'activity' ? undefined : (messageActionItems(m) as ContextMenuItem[][])"
                    :disabled="bubbleKind(m) === 'deleted' || bubbleKind(m) === 'activity'"
                  >
                    <div :class="['group/bubble relative', bubbleClass(m)]">
                      <UDropdownMenu
                        v-if="bubbleKind(m) !== 'deleted' && bubbleKind(m) !== 'activity'"
                        :items="messageActionItems(m)"
                        :open="isMenuOpen(m.id)"
                        :content="{ align: messageSide(m) === 'right' ? 'end' : 'start' }"
                        @update:open="v => toggleMenu(m.id, v)"
                      >
                        <button
                          type="button"
                          class="absolute right-1.5 top-1.5 z-10 grid size-5 place-content-center rounded-full text-current transition-opacity duration-150"
                          :class="[
                            isMenuOpen(m.id) ? 'opacity-100' : 'opacity-0 group-hover/bubble:opacity-90 hover:!opacity-100'
                          ]"
                          :aria-label="t('conversations.message.actions.more')"
                        >
                          <UIcon name="i-lucide-chevron-down" class="size-4" />
                        </button>
                      </UDropdownMenu>

                      <div
                        v-if="quotedReply(m)"
                        class="mb-1.5 border-l-2 border-current/40 bg-black/10 px-2 py-1 text-[11px] leading-tight opacity-80"
                      >
                        <div class="font-medium">
                          {{ quotedReply(m)?.author ?? t('conversations.message.actions.reply') }}
                        </div>
                        <div class="line-clamp-2 whitespace-pre-wrap break-words text-xs">
                          {{ quotedReply(m)?.content || t('conversations.message.actions.attachment') }}
                        </div>
                      </div>

                      <template v-if="bubbleKind(m) === 'deleted'">
                        <p class="font-semibold">
                          {{ t('conversations.message.deleted') }}
                        </p>
                      </template>

                      <template v-else-if="bubbleKind(m) === 'private'">
                        <div
                          v-if="m.content && isMarkdown(m)"
                          class="markdown-body"
                          v-html="renderMarkdown(m.content ?? '')"
                        /><!-- eslint-disable-line vue/no-v-html -->
                        <p v-else-if="m.content">
                          {{ m.content }}
                        </p>
                      </template>

                      <template v-else-if="bubbleKind(m) === 'activity'">
                        <p>
                          {{ m.content }}
                        </p>
                      </template>

                      <template v-else-if="bubbleKind(m) === 'error'">
                        <p class="text-error">
                          {{ m.content }}
                        </p>
                      </template>

                      <template v-else-if="m.content && isMarkdown(m)">
                        <!-- eslint-disable-next-line vue/no-v-html -->
                        <div class="markdown-body" v-html="renderMarkdown(m.content ?? '')" />
                      </template>

                      <template v-else-if="m.content">
                        <p class="whitespace-pre-wrap">
                          {{ m.content }}
                        </p>
                      </template>

                      <div v-if="hasAttachments(m)" class="mt-1 flex flex-col gap-2">
                        <template v-for="(att, ai) in getAttachments(m)" :key="ai">
                          <ConversationsAudioPlayer
                            v-if="isAudioAttachment(att)"
                            :path="att.path"
                            :src="att.fileUrl"
                            :variant="messageSide(m) === 'right' ? 'outgoing' : 'incoming'"
                            :track-id="`msg:${m.id}:${ai}`"
                            :account-id="conversation.accountId"
                            :conversation-id="conversation.id"
                            :title="contactName"
                          />
                          <a
                            v-else-if="att.fileUrl"
                            :href="att.fileUrl"
                            target="_blank"
                            class="inline-flex items-center gap-1.5 rounded-md bg-default px-2 py-1 text-xs text-primary ring ring-default hover:underline"
                          >
                            <UIcon name="i-lucide-paperclip" class="size-3" />
                            {{ att.fileType || 'attachment' }}
                          </a>
                          <span
                            v-else
                            class="inline-flex items-center gap-1.5 rounded-md bg-default px-2 py-1 text-xs text-muted ring ring-default"
                          >
                            <UIcon name="i-lucide-paperclip" class="size-3" />
                            {{ att.fileType || 'attachment' }}
                          </span>
                        </template>
                      </div>

                      <div
                        v-if="bubbleKind(m) !== 'activity'"
                        class="-mb-0.5 mt-1 flex items-center justify-end gap-1 text-[10px] leading-none opacity-70"
                      >
                        <span class="tabular-nums">{{ messageTime(m) }}</span>
                        <UIcon
                          v-if="m.messageType === 1"
                          :name="messageStatusDisplay(m, t).icon"
                          :class="['size-3', messageStatusDisplay(m, t).color]"
                        />
                      </div>
                    </div>
                  </UContextMenu>
                </template>
              </UChatMessage>
            </UChatMessages>
          </div>

          <ConversationsConversationComposer :conversation="conversation" />
        </div>
      </section>

      <aside
        v-if="detailsOpen"
        class="hidden w-72 shrink-0 flex-col border-l border-default bg-default lg:flex xl:w-80"
      >
        <div class="flex h-14 shrink-0 items-center justify-between border-b border-default px-4">
          <h2 class="text-sm font-semibold text-highlighted">
            {{ t('conversations.detail.contacts') }}
          </h2>
          <UButton
            icon="i-lucide-x"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('common.close')"
            @click="detailsOpen = false"
          />
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto">
          <section class="border-b border-default px-4 py-3">
            <div class="flex flex-col items-center text-center">
              <div class="relative">
                <UAvatar
                  :alt="contactName"
                  :src="contactAvatar"
                  size="xl"
                />
                <span class="absolute -bottom-1 -right-1 rounded-md bg-elevated px-1 py-0.5 text-[10px] font-semibold text-muted ring ring-default">
                  {{ channelBadge }}
                </span>
              </div>

              <p class="mt-2 truncate max-w-full text-base font-semibold text-highlighted">
                {{ contactName }}
              </p>
              <p v-if="contactIdentifier" class="truncate max-w-full text-xs text-muted">
                {{ contactIdentifier }}
              </p>

              <div class="mt-2 flex flex-wrap items-center justify-center gap-1.5">
                <UBadge
                  :label="statusLabel"
                  :color="statusColor"
                  variant="subtle"
                  size="xs"
                />
                <UBadge
                  :label="`#${conversation.displayId}`"
                  color="neutral"
                  variant="soft"
                  size="xs"
                />
                <UBadge
                  v-if="conversation.inbox?.name"
                  :label="conversation.inbox.name"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  class="max-w-full truncate"
                />
              </div>
            </div>

            <div class="mt-3 grid grid-cols-4 gap-1.5">
              <UTooltip :text="t('conversations.detail.message')">
                <UButton
                  icon="i-lucide-message-circle"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  block
                  :aria-label="t('conversations.detail.message')"
                />
              </UTooltip>
              <UTooltip :text="t('common.edit')">
                <UButton
                  icon="i-lucide-pencil"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  block
                  :aria-label="t('common.edit')"
                />
              </UTooltip>
              <UTooltip :text="t('conversations.compose.voice')">
                <UButton
                  icon="i-lucide-mic"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  block
                  :aria-label="t('conversations.compose.voice')"
                />
              </UTooltip>
              <UTooltip :text="t('common.delete')">
                <UButton
                  icon="i-lucide-trash-2"
                  color="error"
                  variant="soft"
                  size="xs"
                  block
                  :aria-label="t('common.delete')"
                />
              </UTooltip>
            </div>

            <dl v-if="contactRows.length" class="mt-3 space-y-1.5">
              <div
                v-for="row in contactRows"
                :key="`${row.icon}-${row.title}`"
                class="flex min-w-0 items-center gap-2"
              >
                <UIcon :name="row.icon" class="size-3.5 shrink-0 text-dimmed" />
                <span class="truncate text-xs text-muted" :title="row.value">{{ row.value }}</span>
              </div>
            </dl>
          </section>

          <UAccordion
            :items="detailSections"
            type="multiple"
            :default-value="['actions']"
            :ui="{
              root: 'space-y-2 p-2',
              item: 'overflow-hidden rounded-md border border-default bg-default last:border-b',
              trigger: 'px-3 py-3 text-sm hover:bg-elevated/60',
              leadingIcon: 'size-4 text-muted',
              body: 'px-3 pb-3 text-sm'
            }"
          >
            <template #trailing="{ open }">
              <UIcon
                name="i-lucide-plus"
                :class="['ms-auto size-4 shrink-0 text-muted transition-transform', open ? 'rotate-45' : '']"
              />
            </template>

            <template #body="{ item }">
              <div v-if="item.value === 'actions'" class="space-y-3">
                <div class="flex items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold text-highlighted">
                    {{ t('conversations.detail.assignedAgent') }}
                  </h3>
                  <UButton
                    :label="t('conversations.detail.assignToMe')"
                    icon="i-lucide-arrow-right"
                    color="primary"
                    variant="link"
                    size="xs"
                    :loading="assignmentLoading"
                    @click="assignToMe"
                  />
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.assignedAgent') }}
                  </p>
                  <UDropdownMenu :items="agentItems" :content="{ align: 'start' }" :disabled="assignmentLoading">
                    <UButton
                      :label="currentAssigneeLabel"
                      trailing-icon="i-lucide-chevron-down"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      block
                      :loading="assignmentLoading"
                      class="justify-between"
                    />
                  </UDropdownMenu>
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.assignedTeam') }}
                  </p>
                  <UDropdownMenu :items="teamItems" :content="{ align: 'start' }" :disabled="assignmentLoading">
                    <UButton
                      :label="currentTeamLabel"
                      trailing-icon="i-lucide-chevron-down"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      block
                      :loading="assignmentLoading"
                      class="justify-between"
                    />
                  </UDropdownMenu>
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.priority') }}
                  </p>
                  <UDropdownMenu :items="priorityItems" :content="{ align: 'start' }">
                    <UButton
                      :label="t('conversations.detail.none')"
                      trailing-icon="i-lucide-chevron-down"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      block
                      class="justify-between"
                    />
                  </UDropdownMenu>
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.labels') }}
                  </p>
                  <div v-if="conversation.labels?.length" class="flex flex-wrap gap-1.5">
                    <span
                      v-for="label in conversation.labels"
                      :key="label"
                      class="rounded-md bg-elevated px-2 py-1 text-xs text-muted"
                    >
                      {{ label }}
                    </span>
                  </div>
                  <UButton
                    v-else
                    :label="t('conversations.detail.addLabels')"
                    icon="i-lucide-plus"
                    color="primary"
                    variant="soft"
                    size="xs"
                  />
                </div>
              </div>

              <div v-else class="flex items-center gap-2 rounded-md bg-elevated/50 px-3 py-2 text-sm text-muted">
                <UIcon name="i-lucide-circle-dashed" class="size-4 shrink-0 text-dimmed" />
                {{ t('conversations.detail.emptySection') }}
              </div>
            </template>
          </UAccordion>
        </div>
      </aside>
    </div>
  </UDashboardPanel>
</template>

<style>
.markdown-body p { margin: 0; }
.markdown-body p + p { margin-top: 0.5rem; }
.markdown-body ul, .markdown-body ol { margin: 0.25rem 0; padding-left: 1.5rem; }
.markdown-body code { background: color-mix(in oklch, currentColor 10%, transparent); padding: 0 0.25rem; border-radius: 0.25rem; font-size: 0.85em; }
.markdown-body pre { background: color-mix(in oklch, currentColor 10%, transparent); padding: 0.5rem; border-radius: 0.375rem; overflow-x: auto; }
.markdown-body a { text-decoration: underline; }
</style>
