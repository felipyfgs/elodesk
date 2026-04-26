<script setup lang="ts">
import type { Message } from '~/stores/messages'
import { messageTime, messageStatusDisplay, messageSide, getAttachments } from '~/utils/chatAdapter'
import { renderMarkdown } from '~/utils/markdown'

const props = withDefaults(defineProps<{
  message: Message
  showSender?: boolean
  senderName?: string
}>(), {
  showSender: false,
  senderName: ''
})

const { t } = useI18n()

const side = computed(() => messageSide(props.message))
const statusDisp = computed(() => messageStatusDisplay(props.message, t))
const attachments = computed(() => getAttachments(props.message))
const hasImageAttachments = computed(() =>
  attachments.value.some(a => a.fileType?.startsWith('image'))
)
</script>

<template>
  <div
    class="flex"
    :class="side === 'right' ? 'justify-end' : 'justify-start'"
  >
    <div
      class="max-w-[75%] space-y-1"
      :class="side === 'right' ? 'items-end' : 'items-start'"
    >
      <p
        v-if="showSender && senderName"
        class="px-1 text-[11px] font-semibold text-muted"
      >
        {{ senderName }}
      </p>

      <div
        class="rounded-2xl px-4 py-2.5 text-sm leading-relaxed"
        :class="side === 'right'
          ? 'bg-primary text-inverted rounded-br-md'
          : 'bg-elevated text-default rounded-bl-md'"
      >
        <div
          v-if="message.content"
          class="whitespace-pre-wrap break-words [&_a]:underline"
          v-html="renderMarkdown(message.content)"
        />

        <div
          v-if="hasImageAttachments"
          class="mt-2 grid gap-2"
        >
          <div
            v-for="att in attachments.filter(a => a.fileType?.startsWith('image'))"
            :key="att.fileUrl ?? att.path"
            class="overflow-hidden rounded-lg"
          >
            <img
              :src="att.fileUrl ?? att.path"
              alt="attachment"
              class="max-w-full object-cover"
              loading="lazy"
            >
          </div>
        </div>

        <div
          v-if="attachments.length > 0 && !hasImageAttachments"
          class="mt-2 space-y-1"
        >
          <UCard
            v-for="att in attachments"
            :key="att.fileUrl ?? att.path"
            :ui="{ root: '!shadow-none !border-default/50', body: '!p-2' }"
          >
            <a
              :href="att.fileUrl ?? att.path"
              target="_blank"
              class="flex items-center gap-2 text-xs font-medium"
            >
              <UIcon name="i-lucide-download" class="size-4 shrink-0" />
              <span class="truncate">{{ att.fileUrl?.split('/').pop() || att.path?.split('/').pop() || t('conversations.message.attachment') }}</span>
            </a>
          </UCard>
        </div>
      </div>

      <div
        class="flex items-center gap-1.5 px-1"
        :class="side === 'right' ? 'justify-end' : 'justify-start'"
      >
        <span class="text-[10px] text-dimmed">
          {{ messageTime(message) }}
        </span>
        <span
          v-if="side === 'right' && statusDisp.icon"
          :class="statusDisp.color"
          :title="statusDisp.label"
        >
          <UIcon :name="statusDisp.icon" class="size-3" />
        </span>
      </div>
    </div>
  </div>
</template>
