<script setup lang="ts">
import type { Conversation } from '~/stores/conversations'
import { resolveContactName, resolveContactIdentifier, resolveContactAvatar } from '~/utils/chatAdapter'

const props = defineProps<{
  conversation: Conversation
}>()

const { t } = useI18n()

const CHANNEL_LABELS: Record<string, string> = {
  'Channel::Api': 'API',
  'Channel::Whatsapp': 'WhatsApp',
  'Channel::Twilio': 'Twilio',
  'Channel::Sms': 'SMS',
  'Channel::Instagram': 'Instagram',
  'Channel::FacebookPage': 'Facebook',
  'Channel::Telegram': 'Telegram',
  'Channel::Line': 'Line',
  'Channel::Tiktok': 'TikTok',
  'Channel::WebWidget': 'Widget',
  'Channel::Email': 'Email',
  'Channel::Twitter': 'X'
}

function channelLabel(): string {
  return CHANNEL_LABELS[props.conversation.inbox?.channelType ?? ''] ?? props.conversation.inbox?.channelType ?? 'CH'
}

const isGroup = computed(() => {
  const attrs = (props.conversation as { additionalAttributes?: Record<string, unknown> }).additionalAttributes
  return !!attrs?.isGroup || !!attrs?.is_group
})
</script>

<template>
  <div class="flex items-center gap-3 border-b border-default px-4 py-3">
    <ChatContactAvatar
      :url="resolveContactAvatar(conversation)"
      :name="resolveContactName(conversation)"
      size="lg"
      :is-group="isGroup"
    />
    <div class="min-w-0 flex-1">
      <h3 class="truncate text-sm font-semibold text-highlighted">
        {{ resolveContactName(conversation) }}
      </h3>
      <p class="truncate text-xs text-muted">
        {{ resolveContactIdentifier(conversation) || t('conversations.detail.noContact') }}
      </p>
    </div>
    <UBadge
      :label="channelLabel()"
      color="neutral"
      variant="soft"
      size="xs"
    />
  </div>
</template>
