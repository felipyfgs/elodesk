<script setup lang="ts">
import { format, isToday, formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import type { ContextMenuItem } from '@nuxt/ui'
import type { Notification } from '~/stores/notifications'
import { channelIcon } from '~/utils/channels'

const props = defineProps<{
  notification: Notification
  selected?: boolean
}>()
const emit = defineEmits<{
  click: [n: Notification]
  toggleRead: [n: Notification]
}>()

const { t } = useI18n()

// Per-type accent — controls the type chip color and the assignment ring
// around the avatar. Same family used elsewhere (assignment vs SLA).
const TYPE_META: Record<string, { icon: string, color: 'primary' | 'warning' | 'error' | 'info' | 'neutral' }> = {
  conversation_assigned: { icon: 'i-lucide-user-plus', color: 'primary' },
  sla_breach: { icon: 'i-lucide-alarm-clock', color: 'warning' }
}

const meta = computed(() => TYPE_META[props.notification.type] ?? { icon: 'i-lucide-bell', color: 'neutral' as const })

const typeLabel = computed(() => {
  const key = `notifications.types.${props.notification.type}Short`
  const translated = t(key)
  return translated === key ? props.notification.type : translated
})

const contactName = computed(() => {
  const conv = props.notification.conversation
  return conv?.contact?.name?.trim() || t('notifications.unknownContact')
})

const contactAvatar = computed(() => props.notification.conversation?.contact?.avatarUrl ?? undefined)

const inboxName = computed(() => props.notification.conversation?.inbox?.name ?? '')

const channelIconName = computed(() => {
  const ch = props.notification.conversation?.inbox?.channelType
  if (!ch) return 'i-lucide-bell'
  return channelIcon(ch)
})

// Mirror ConversationsList.lastMessage(): pick a directional icon based on
// message type and a fallback for activity/private. Falls back to the type
// label when the conversation summary isn't attached (deleted convo, etc).
const lastMessage = computed(() => {
  const msg = props.notification.conversation?.lastMessage
  if (!msg) {
    const p = props.notification.payload as Record<string, unknown> | undefined
    const text = (p?.body as string | undefined) ?? (p?.message as string | undefined) ?? ''
    return { text, icon: meta.value.icon, isPrivate: false }
  }
  const isPrivate = !!msg.private
  const isOutgoing = msg.messageType === 1
  const isActivity = msg.messageType === 2
  let icon = 'i-lucide-corner-up-right'
  if (isPrivate) icon = 'i-lucide-lock'
  else if (isActivity) icon = 'i-lucide-info'
  else if (isOutgoing) icon = 'i-lucide-corner-up-right'
  else icon = 'i-lucide-arrow-left'
  return { text: msg.content?.trim() ?? '', icon, isPrivate }
})

// Same compact format as conversations: HH:mm if today, "X days" otherwise.
const timeLabel = computed(() => {
  const d = new Date(props.notification.createdAt)
  if (Number.isNaN(d.getTime())) return ''
  if (isToday(d)) return format(d, 'HH:mm')
  return formatDistanceToNow(d, { addSuffix: false, locale: ptBR })
})

const isUnread = computed(() => !props.notification.readAt)

// "Órfã" = backend não conseguiu hidratar a conversa (deletada). Exibimos
// como item esmaecido para o agente entender que clicar não vai abrir a
// thread — só mostra o estado "Conversa removida" no painel direito.
const isOrphan = computed(() => !props.notification.conversation)

// Right-click menu: only the read/unread toggle for now. Same affordance the
// conversations list offers via ContextMenu.
const contextItems = computed<ContextMenuItem[][]>(() => [[
  isUnread.value
    ? { label: t('notifications.markRead'), icon: 'i-lucide-mail-open', onSelect: () => emit('toggleRead', props.notification) }
    : { label: t('notifications.markUnread'), icon: 'i-lucide-mail', onSelect: () => emit('toggleRead', props.notification) }
]])

function onClick() {
  emit('click', props.notification)
}
</script>

<template>
  <UContextMenu :items="contextItems">
    <div
      class="group relative grid cursor-pointer grid-cols-[auto_minmax(0,1fr)] items-center gap-3 border-l-2 px-3 py-3 transition-colors outline-none"
      :class="[
        selected
          ? 'border-primary bg-primary/10'
          : 'border-bg hover:border-primary hover:bg-primary/5 focus-visible:border-primary focus-visible:bg-primary/5',
        isOrphan ? 'opacity-60' : ''
      ]"
      role="option"
      tabindex="0"
      :aria-selected="selected"
      :aria-label="t('notifications.openLabel', { name: contactName })"
      @click="onClick"
      @keydown.enter.prevent="onClick"
      @keydown.space.prevent="onClick"
    >
      <!-- Avatar + channel chip overlay (same compose as ConversationsList) -->
      <div class="relative size-8 shrink-0">
        <UAvatar
          :alt="contactName"
          :src="contactAvatar"
          size="md"
          class="size-8"
        />
        <span class="absolute -bottom-1 -right-1 flex size-5 items-center justify-center rounded-md bg-default ring ring-default">
          <UIcon :name="channelIconName" class="size-3 text-muted" />
        </span>
      </div>

      <div class="min-w-0">
        <div class="flex min-w-0 items-start gap-2">
          <div class="min-w-0 flex-1">
            <h4
              class="truncate text-sm leading-5"
              :class="isUnread ? 'font-semibold text-highlighted' : 'font-medium text-default'"
            >
              {{ contactName }}
            </h4>
            <div class="mt-0.5 flex min-w-0 items-center gap-1.5 text-[11px] text-muted">
              <span v-if="notification.conversation" class="shrink-0 font-medium">
                #{{ notification.conversation.displayId }}
              </span>
              <span v-if="notification.conversation && inboxName" class="text-dimmed">/</span>
              <span v-if="inboxName" class="truncate">{{ inboxName }}</span>
            </div>
          </div>

          <div class="flex shrink-0 flex-col items-end gap-1">
            <span class="text-[11px] text-muted">
              {{ timeLabel }}
            </span>
            <span
              v-if="isUnread"
              class="size-2 rounded-full bg-primary"
              :aria-label="t('notifications.unreadIndicator')"
            />
          </div>
        </div>

        <div v-if="lastMessage.text" class="mt-2 flex min-w-0 items-center gap-1.5">
          <UIcon
            :name="lastMessage.icon"
            class="size-3.5 shrink-0"
            :class="lastMessage.isPrivate ? 'text-warning' : 'text-dimmed'"
          />
          <p
            class="min-w-0 flex-1 truncate text-xs"
            :class="isUnread ? 'text-default font-medium' : 'text-muted'"
          >
            {{ lastMessage.text }}
          </p>
        </div>

        <!-- Type chip — small accent so SLA and assignments are scannable -->
        <div class="mt-1.5 flex items-center gap-1">
          <span
            class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px]"
            :class="{
              'bg-primary/10 text-primary': meta.color === 'primary',
              'bg-warning/10 text-warning': meta.color === 'warning',
              'bg-error/10 text-error': meta.color === 'error',
              'bg-info/10 text-info': meta.color === 'info',
              'bg-elevated text-muted': meta.color === 'neutral'
            }"
          >
            <UIcon :name="meta.icon" class="size-3" />
            {{ typeLabel }}
          </span>
        </div>
      </div>
    </div>
  </UContextMenu>
</template>
