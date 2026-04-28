<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { Conversation } from '~/stores/conversations'
import {
  resolveContactName,
  resolveContactIdentifier,
  resolveContactAvatar
} from '~/utils/chatAdapter'

const props = withDefaults(defineProps<{
  conversation: Conversation
  showBack?: boolean
  detailsOpen: boolean
}>(), {
  showBack: false
})

const emit = defineEmits<{
  'back': []
  'update:detailsOpen': [value: boolean]
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

function toggleDetails() {
  emit('update:detailsOpen', !props.detailsOpen)
}
</script>

<template>
  <header class="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-default px-3 sm:px-4">
    <div class="flex min-w-0 flex-1 items-center gap-2">
      <UTooltip v-if="showBack" :text="t('common.close')">
        <UButton
          icon="i-lucide-arrow-left"
          color="neutral"
          variant="ghost"
          size="sm"
          :aria-label="t('common.close')"
          @click="emit('back')"
        />
      </UTooltip>

      <div class="flex min-w-0 flex-1 items-center gap-3 border-b border-default px-4 py-3">
        <ConversationsContactAvatar
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
    </div>

    <div class="flex shrink-0 items-center gap-1">
      <ConversationsThreadStatusActions :conversation="conversation" />
      <UTooltip :text="t('conversations.detail.contacts')">
        <UButton
          icon="i-lucide-user-round"
          color="neutral"
          :variant="detailsOpen ? 'soft' : 'ghost'"
          size="xs"
          class="hidden lg:inline-flex"
          :aria-label="t('conversations.detail.contacts')"
          @click="toggleDetails"
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
</template>
