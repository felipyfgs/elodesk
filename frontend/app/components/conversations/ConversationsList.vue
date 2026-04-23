<script setup lang="ts">
import { format, isToday, formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import type { Conversation } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'

const props = defineProps<{
  items: Conversation[]
}>()

const selected = defineModel<Conversation | null>()
const convs = useConversationsStore()

const itemRefs = ref<Record<string, Element | null>>({})
const hoveredId = ref<string | null>(null)

const CHANNEL_ICONS: Record<string, string> = {
  'Channel::Api': 'i-lucide-webhook',
  'Channel::Whatsapp': 'i-simple-icons-whatsapp',
  'Channel::Twilio': 'i-lucide-cloud',
  'Channel::Sms': 'i-lucide-message-square',
  'Channel::Instagram': 'i-simple-icons-instagram',
  'Channel::FacebookPage': 'i-simple-icons-facebook',
  'Channel::Telegram': 'i-simple-icons-telegram',
  'Channel::Line': 'i-simple-icons-line',
  'Channel::Tiktok': 'i-simple-icons-tiktok',
  'Channel::WebWidget': 'i-lucide-globe',
  'Channel::Email': 'i-lucide-mail',
  'Channel::Twitter': 'i-simple-icons-x'
}

function contactName(c: Conversation): string {
  return c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}

function contactAvatar(c: Conversation): string | undefined {
  return c.meta?.sender?.thumbnail ?? c.contactInbox?.contact?.avatarUrl ?? undefined
}

function unreadCount(c: Conversation): number {
  return c.meta?.unreadCount ?? 0
}

function hasUnread(c: Conversation): boolean {
  return unreadCount(c) > 0
}

function channelIcon(c: Conversation): string {
  return CHANNEL_ICONS[c.channelType] ?? 'i-lucide-inbox'
}

function lastMessage(c: Conversation): { text: string, icon: string, isPrivate: boolean } {
  const msg = c.meta?.lastNonActivityMessage
  if (!msg) return { text: '', icon: 'i-lucide-info', isPrivate: false }

  const isPrivate = !!(msg as Record<string, unknown>).private
  const isOutgoing = msg.messageType === 1
  const isActivity = msg.messageType === 2

  let icon = 'i-lucide-arrow-reply'
  if (isPrivate) icon = 'i-lucide-lock'
  else if (isActivity) icon = 'i-lucide-info'
  else if (isOutgoing) icon = 'i-lucide-arrow-reply'
  else icon = 'i-lucide-arrow-left'

  let text = msg.content || ''
  if (!text && msg.attachments?.length) text = `[${msg.attachments.length} anexo(s)]`
  if (!text) text = ''

  return { text, icon, isPrivate }
}

function timeLabel(c: Conversation): string {
  const d = new Date(c.lastActivityAt)
  if (isToday(d)) return format(d, 'HH:mm')
  return formatDistanceToNow(d, { addSuffix: false, locale: ptBR })
}

function isActive(c: Conversation): boolean {
  return selected.value?.id === c.id
}

watch(selected, () => {
  if (!selected.value) return
  const ref = itemRefs.value[selected.value.id]
  if (ref) ref.scrollIntoView({ block: 'nearest' })
})

defineShortcuts({
  arrowdown: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    if (idx === -1) selected.value = props.items[0] ?? null
    else if (idx < props.items.length - 1) selected.value = props.items[idx + 1] ?? null
  },
  arrowup: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    if (idx === -1) selected.value = props.items.at(-1) ?? null
    else if (idx > 0) selected.value = props.items[idx - 1] ?? null
  }
})
</script>

<template>
  <div class="overflow-y-auto">
    <div
      v-for="c in items"
      :key="c.id"
      :ref="(el) => { itemRefs[c.id] = el as Element | null }"
      class="relative flex items-start gap-3 px-3 py-3 cursor-pointer transition-colors border-b border-default"
      :class="[
        isActive(c)
          ? 'bg-elevated'
          : 'hover:bg-elevated/50'
      ]"
      @click="selected = c"
      @mouseenter="hoveredId = c.id"
      @mouseleave="hoveredId = null"
    >
      <!-- Avatar with checkbox overlay -->
      <div class="relative shrink-0 mt-0.5">
        <UAvatar
          :alt="contactName(c)"
          :src="contactAvatar(c)"
          size="md"
        />
        <!-- Checkbox overlay on hover -->
        <label
          v-if="hoveredId === c.id || convs.selection.includes(c.id)"
          class="absolute inset-0 z-10 flex items-center justify-center rounded-full cursor-pointer bg-default/60 backdrop-blur-sm"
          @click.stop
        >
          <UCheckbox
            :model-value="convs.selection.includes(c.id)"
            @update:model-value="() => convs.toggleSelection(c.id)"
          />
        </label>
      </div>

      <!-- Content -->
      <div class="flex-1 min-w-0">
        <!-- Row 1: Channel icon + Name + Timestamp -->
        <div class="flex items-center gap-1.5">
          <UIcon :name="channelIcon(c)" class="size-3 shrink-0 text-muted" />
          <span class="text-xs text-muted truncate max-w-[100px]">
            {{ c.inbox?.name }}
          </span>
          <span class="flex-1" />
          <span class="text-[10px] text-muted shrink-0">
            {{ timeLabel(c) }}
          </span>
          <!-- Unread badge -->
          <span
            v-if="hasUnread(c)"
            class="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-primary text-[10px] font-semibold text-inverted"
          >
            {{ unreadCount(c) > 99 ? '99+' : unreadCount(c) }}
          </span>
        </div>

        <!-- Row 2: Contact name -->
        <h4
          class="text-sm truncate mt-0.5"
          :class="hasUnread(c) ? 'font-semibold text-highlighted' : 'font-medium text-default'"
        >
          {{ contactName(c) }}
        </h4>

        <!-- Row 3: Message preview -->
        <div v-if="lastMessage(c).text" class="flex items-center gap-1.5 mt-0.5">
          <UIcon :name="lastMessage(c).icon" class="size-3 shrink-0 text-dimmed" />
          <p
            class="text-xs truncate flex-1 min-w-0"
            :class="hasUnread(c) ? 'text-default font-medium' : 'text-muted'"
          >
            {{ lastMessage(c).text }}
          </p>
        </div>
        <p v-else class="text-xs text-dimmed mt-0.5 italic">
          {{ $t('conversations.empty') }}
        </p>

        <!-- Row 4: Labels -->
        <div v-if="c.labels?.length" class="flex items-center gap-1 mt-1 flex-wrap">
          <span
            v-for="label in c.labels.slice(0, 3)"
            :key="label.id"
            class="inline-flex items-center text-[10px] rounded-full px-2 py-0.5 font-medium truncate max-w-[80px]"
            :style="{ backgroundColor: label.color + '18', color: label.color }"
          >
            {{ label.title }}
          </span>
          <span v-if="c.labels.length > 3" class="text-[10px] text-muted">
            +{{ c.labels.length - 3 }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
