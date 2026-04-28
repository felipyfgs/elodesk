<script setup lang="ts">
import { format, isToday, formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import type { Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { resolveContactName, resolveContactAvatar } from '~/utils/chatAdapter'

const props = defineProps<{
  conversations: Conversation[]
}>()

const { t } = useI18n()
const auth = useAuthStore()
const aid = computed(() => auth.account?.id ?? '')

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

function statusColor(status: number): 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral' {
  const map: Record<number, 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'> = {
    0: 'success',
    1: 'neutral',
    2: 'warning',
    3: 'info'
  }
  return map[status] ?? 'neutral'
}

const STATUS_LABELS: Record<number, string> = {
  0: 'OPEN',
  1: 'RESOLVED',
  2: 'PENDING',
  3: 'SNOOZED'
}

function channelIcon(c: Conversation): string {
  return CHANNEL_ICONS[c.inbox?.channelType ?? ''] ?? 'i-lucide-inbox'
}

function lastMessagePreview(c: Conversation): { text: string, icon: string } {
  const msg = c.lastNonActivityMessage
  if (!msg) return { text: '—', icon: 'i-lucide-message-circle' }

  const isPrivate = !!msg.private
  const isOutgoing = msg.messageType === 1

  let icon = 'i-lucide-arrow-left'
  if (isPrivate) icon = 'i-lucide-lock'
  else if (isOutgoing) icon = 'i-lucide-arrow-up-right'

  let text = msg.content || ''
  if (!text && msg.attachments?.length) text = `[${msg.attachments.length} anexo(s)]`
  if (!text) text = '—'

  return { text, icon }
}

function timeLabel(c: Conversation): string {
  if (!c.lastActivityAt) return ''
  const d = new Date(c.lastActivityAt)
  if (Number.isNaN(d.getTime())) return ''
  if (isToday(d)) return format(d, 'HH:mm')
  return formatDistanceToNow(d, { addSuffix: false, locale: ptBR })
}

const sorted = computed(() =>
  [...props.conversations].sort((a, b) => {
    const ta = new Date(a.lastActivityAt).getTime() || 0
    const tb = new Date(b.lastActivityAt).getTime() || 0
    return tb - ta
  })
)
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-medium text-highlighted">
      {{ t('contactDetail.sidebar.history') }}
    </h3>

    <p v-if="!sorted.length" class="text-sm text-muted">
      {{ t('common.noResults') }}
    </p>

    <ul v-else class="flex flex-col gap-1.5">
      <li v-for="conv in sorted" :key="conv.id">
        <NuxtLink
          :to="aid ? `/accounts/${aid}/conversations/${conv.id}` : `/conversations/${conv.id}`"
          class="block rounded-md border border-default px-3 py-2.5 hover:bg-elevated transition-colors"
        >
          <div class="flex items-start gap-2.5">
            <div class="relative shrink-0">
              <UAvatar
                :alt="resolveContactName(conv) || `#${conv.displayId ?? conv.id}`"
                :src="resolveContactAvatar(conv)"
                size="sm"
              />
              <span class="absolute -bottom-0.5 -right-0.5 flex size-4 items-center justify-center rounded-full bg-default ring ring-default">
                <UIcon :name="channelIcon(conv)" class="size-2.5 text-muted" />
              </span>
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-2">
                <span class="truncate text-sm font-medium text-default">
                  {{ resolveContactName(conv) || `#${conv.displayId ?? conv.id}` }}
                </span>
                <span class="text-[11px] text-dimmed shrink-0">{{ timeLabel(conv) }}</span>
              </div>

              <div class="mt-0.5 flex items-center gap-1.5 text-xs text-muted">
                <UIcon :name="lastMessagePreview(conv).icon" class="size-3 shrink-0" />
                <span class="truncate">{{ lastMessagePreview(conv).text }}</span>
              </div>

              <div class="mt-1.5 flex items-center justify-between gap-2">
                <span class="text-[11px] text-dimmed truncate">
                  {{ conv.inbox?.name ?? '' }} · #{{ conv.displayId ?? conv.id }}
                </span>
                <UBadge :color="statusColor(conv.status)" variant="subtle" size="xs">
                  {{ STATUS_LABELS[conv.status] ?? conv.status }}
                </UBadge>
              </div>
            </div>
          </div>
        </NuxtLink>
      </li>
    </ul>
  </div>
</template>
