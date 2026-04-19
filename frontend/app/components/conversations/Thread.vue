<script setup lang="ts">
import { format } from 'date-fns'
import { useMessagesStore, type Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'

const props = defineProps<{
  conversation: Conversation
}>()
const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const messages = useMessagesStore()

const list = computed<Message[]>(() => messages.byConversation[props.conversation.id] ?? [])
const reply = ref('')
const sending = ref(false)

async function loadMessages() {
  if (!auth.account?.id) return
  const res = await api<Message[]>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`)
  messages.set(props.conversation.id, [...res].reverse())
}

async function send() {
  if (!auth.account?.id || !reply.value) return
  sending.value = true
  try {
    await api(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`, {
      method: 'POST',
      body: { body: reply.value }
    })
    reply.value = ''
  } finally {
    sending.value = false
  }
}

watch(() => props.conversation.id, loadMessages, { immediate: true })

function title(c: Conversation) {
  return c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}
</script>

<template>
  <UDashboardPanel id="conversation-thread">
    <UDashboardNavbar :title="title(conversation)" :toggle="false">
      <template #leading>
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          class="-ms-1.5"
          @click="emit('close')"
        />
      </template>
    </UDashboardNavbar>

    <div class="flex flex-col sm:flex-row justify-between gap-1 p-4 sm:px-6 border-b border-default">
      <div class="flex items-start gap-4 sm:my-1.5">
        <UAvatar :alt="title(conversation)" size="3xl" />
        <div class="min-w-0">
          <p class="font-semibold text-highlighted">
            {{ title(conversation) }}
          </p>
          <p class="text-muted text-sm">
            {{ conversation.contactInbox?.contact?.waJid ?? '—' }}
          </p>
        </div>
      </div>
      <p class="max-sm:pl-16 text-muted text-sm sm:mt-2">
        {{ conversation.inbox?.name }}
      </p>
    </div>

    <div class="flex-1 overflow-y-auto p-4 sm:p-6 flex flex-col gap-2">
      <div
        v-for="m in list"
        :key="m.id"
        :class="[
          'max-w-[80%] rounded-lg px-3 py-2 text-sm',
          m.messageType === 'OUTGOING' ? 'self-end bg-primary/90 text-white' : 'self-start bg-elevated'
        ]"
      >
        <p v-if="(m.contentAttributes as Record<string, unknown>)?.deleted" class="italic text-muted">
          {{ t('conversations.message.deleted') }}
        </p>
        <p v-else class="whitespace-pre-wrap">
          {{ m.content }}
          <span v-if="(m.contentAttributes as Record<string, unknown>)?.edited" class="text-[10px] opacity-60 ms-1">
            ({{ t('conversations.message.edited') }})
          </span>
        </p>
        <div class="flex justify-between items-center gap-3 mt-1">
          <span class="text-[10px] opacity-60">
            {{ format(new Date(m.createdAt), 'HH:mm') }}
          </span>
          <span class="text-[10px] opacity-60">
            {{ t(`conversations.message.status.${m.status}`) }}
          </span>
        </div>
      </div>
    </div>

    <div class="pb-4 px-4 sm:px-6 shrink-0">
      <UCard variant="subtle" class="mt-auto" :ui="{ header: 'flex items-center gap-1.5 text-dimmed' }">
        <template #header>
          <UIcon name="i-lucide-reply" class="size-5" />
          <span class="text-sm truncate">{{ t('conversations.compose.placeholder') }}</span>
        </template>

        <form @submit.prevent="send">
          <UTextarea
            v-model="reply"
            color="neutral"
            variant="none"
            required
            autoresize
            :placeholder="t('conversations.compose.placeholder')"
            :rows="3"
            :disabled="sending"
            class="w-full"
            :ui="{ base: 'p-0 resize-none' }"
          />
          <div class="flex items-center justify-end gap-2">
            <UButton
              type="submit"
              color="primary"
              :loading="sending"
              :label="t('conversations.compose.send')"
              icon="i-lucide-send"
            />
          </div>
        </form>
      </UCard>
    </div>
  </UDashboardPanel>
</template>
