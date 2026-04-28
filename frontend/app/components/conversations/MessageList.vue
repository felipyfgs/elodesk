<script setup lang="ts">
import type { Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'
import {
  messageRole,
  messageVariant,
  messageSide,
  messageBubbleKind,
  messageParts,
  shouldGroupWith
} from '~/utils/chatAdapter'

const props = defineProps<{
  messages: Message[]
  conversation: Conversation
}>()

const { t } = useI18n()

function isGrouped(index: number): boolean {
  if (index === 0) return false
  const prev = props.messages[index - 1]
  const curr = props.messages[index]
  if (!prev || !curr) return false
  return shouldGroupWith(prev, curr)
}

function messageUi(m: Message) {
  const kind = messageBubbleKind(m)
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
</script>

<template>
  <UChatMessages
    class="mx-auto w-full max-w-4xl px-0 py-4"
    :should-scroll-to-bottom="false"
    :auto-scroll="false"
    :spacing-offset="0"
  >
    <div v-if="!messages.length" class="flex flex-col items-center justify-center py-12 text-muted">
      <UIcon name="i-lucide-message-circle-off" class="mb-2 size-8 text-dimmed" />
      <p class="text-sm">
        {{ t('conversations.thread.empty') }}
      </p>
    </div>

    <UChatMessage
      v-for="(m, i) in messages"
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
        <ConversationsMessageBubble
          :message="m"
          :conversation="conversation"
          :grouped="isGrouped(i)"
        />
      </template>
    </UChatMessage>
  </UChatMessages>
</template>
