<script setup lang="ts">
import type { Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore } from '~/stores/messages'

const props = defineProps<{
  conversation: Conversation
  mode: 'reply' | 'private'
  charCount: number
  maxChars: number
}>()

const emit = defineEmits<{
  'update:mode': [value: 'reply' | 'private']
}>()

const { t } = useI18n()
const auth = useAuthStore()
const messages = useMessagesStore()

const charExceeded = computed(() => props.maxChars > 0 && props.charCount > props.maxChars)

const replyTarget = computed(() => messages.replyingTo[props.conversation.id] ?? null)
const replyAuthor = computed(() => {
  const r = replyTarget.value
  if (!r) return ''
  const isAgent = r.sender?.type === 'user' || r.senderType === 'USER'
  if (isAgent) return auth.user?.name ?? t('conversations.message.actions.reply')
  return props.conversation.meta?.sender?.name ?? t('conversations.message.actions.reply')
})

function cancelReply() {
  messages.clearReplyTarget(props.conversation.id)
}

function setMode(value: 'reply' | 'private') {
  emit('update:mode', value)
}
</script>

<template>
  <div>
    <div class="mb-2 flex items-center justify-between gap-2">
      <div
        class="inline-flex shrink-0 items-center gap-0.5 rounded-md p-0.5 ring transition-colors"
        :class="mode === 'private'
          ? 'bg-warning/10 ring-warning/25'
          : 'bg-default ring-default'"
      >
        <UButton
          :label="t('conversations.compose.replyTab')"
          color="neutral"
          :variant="mode === 'reply' ? 'soft' : 'ghost'"
          size="xs"
          @click="setMode('reply')"
        />
        <UButton
          :label="t('conversations.compose.privateTab')"
          :color="mode === 'private' ? 'warning' : 'neutral'"
          :variant="mode === 'private' ? 'soft' : 'ghost'"
          size="xs"
          @click="setMode('private')"
        />
      </div>
      <span
        v-if="maxChars > 0"
        class="truncate text-xs"
        :class="charExceeded ? 'font-medium text-error' : 'text-dimmed'"
      >
        {{ charCount }}/{{ maxChars }}
      </span>
    </div>

    <div
      v-if="replyTarget"
      class="mb-2 flex items-start gap-2 rounded-md border-l-2 border-primary bg-elevated/70 px-2 py-1 ring ring-default"
    >
      <UIcon name="i-lucide-reply" class="mt-0.5 size-3.5 shrink-0 text-primary" />
      <div class="min-w-0 flex-1">
        <div class="text-[11px] font-medium text-primary">
          {{ replyAuthor }}
        </div>
        <div class="line-clamp-2 whitespace-pre-wrap break-words text-xs text-muted">
          {{ replyTarget.content || t('conversations.message.actions.attachment') }}
        </div>
      </div>
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('common.close')"
        @click="cancelReply"
      />
    </div>
  </div>
</template>
