<script setup lang="ts">
import type { Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'
import {
  messageRole,
  messageVariant,
  messageSide,
  messageBubbleKind,
  messageParts,
  messageIsForwardable,
  shouldGroupWith
} from '~/utils/chatAdapter'
import { forwardSelectionModeKey, forwardSelectedIdsKey } from '~/utils/forward'

const props = defineProps<{
  messages: Message[]
  conversation: Conversation
}>()

const { t } = useI18n()

const _selectionModeRef = inject(forwardSelectionModeKey, null)
const _selectedIdsRef = inject(forwardSelectedIdsKey, null)
const selectionMode = computed(() => _selectionModeRef?.value ?? false)
const selectedIds = computed(() => _selectedIdsRef?.value ?? new Set<string>())

const MAX_SELECTION = 5

function isSelected(m: Message): boolean {
  return selectedIds.value.has(String(m.id))
}

function isMaxReached(m: Message): boolean {
  return selectedIds.value.size >= MAX_SELECTION && !isSelected(m)
}

function toggle(m: Message) {
  if (!_selectedIdsRef) return
  if (!messageIsForwardable(m)) return
  const ids = new Set(_selectedIdsRef.value)
  const key = String(m.id)
  if (ids.has(key)) {
    ids.delete(key)
  } else if (ids.size < MAX_SELECTION) {
    ids.add(key)
  }
  _selectedIdsRef.value = ids
}

function onRowClick(m: Message) {
  if (!selectionMode.value) return
  if (!messageIsForwardable(m)) return
  toggle(m)
}

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
  <!--
    `!flex-none` sobrescreve o `flex-1` default do slot root do
    UChatMessages (definido em @nuxt/ui chat-messages.ts). Sem isso, o
    componente cresce até preencher 100% da altura do scroll container,
    anulando o `justify-end` do wrapper em Thread.vue — que é o que faz
    poucas mensagens grudarem no fundo, próximas ao composer.
  -->
  <UChatMessages
    class="mx-auto w-full max-w-5xl xl:max-w-6xl 2xl:max-w-7xl px-0 py-4 !flex-none"
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

    <div
      v-for="(m, i) in messages"
      :key="m.id"
      class="flex items-center transition-colors"
      :class="[
        selectionMode && messageIsForwardable(m) ? 'cursor-pointer' : '',
        selectionMode && isSelected(m) ? 'bg-primary/10' : '',
        selectionMode && messageIsForwardable(m) && !isSelected(m) ? 'hover:bg-elevated/40' : ''
      ]"
      @click="onRowClick(m)"
    >
      <div
        v-if="selectionMode"
        class="w-10 shrink-0 self-stretch flex items-center justify-center pb-4"
      >
        <UCheckbox
          v-if="messageIsForwardable(m)"
          :model-value="isSelected(m)"
          :disabled="isMaxReached(m)"
          :ui="{ base: 'size-4' }"
          :aria-label="t('conversations.forward.triggerAction')"
          @click.stop
          @update:model-value="toggle(m)"
        />
      </div>

      <div class="flex-1 min-w-0">
        <UChatMessage
          :id="String(m.id)"
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
      </div>
    </div>
  </UChatMessages>
</template>
