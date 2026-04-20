<script setup lang="ts">
import { format } from 'date-fns'
import { useAuthStore } from '~/stores/auth'
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

async function loadMessages() {
  if (!auth.account?.id) return
  const res = await api<{ payload: Message[] }>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`)
  if (res.payload) {
    messages.set(props.conversation.id, [...res.payload].reverse())
  }
}

watch(() => props.conversation.id, loadMessages, { immediate: true })

function contactName(c: Conversation): string {
  return c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}

function contactIdentifier(c: Conversation): string {
  return c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}

function messageRole(m: Message): 'user' | 'assistant' | 'system' {
  if (m.messageType === 'ACTIVITY' || m.messageType === 'TEMPLATE') return 'system'
  if (m.messageType === 'OUTGOING') return 'user'
  return 'assistant'
}

const statusItems = computed(() => {
  const items: { label: string, icon: string, status: string }[] = []
  if (props.conversation.status !== 'OPEN') {
    items.push({ label: t('conversations.actions.open'), icon: 'i-lucide-message-circle', status: 'OPEN' })
  }
  if (props.conversation.status !== 'PENDING') {
    items.push({ label: t('conversations.actions.pending'), icon: 'i-lucide-clock', status: 'PENDING' })
  }
  if (props.conversation.status !== 'RESOLVED') {
    items.push({ label: t('conversations.actions.resolve'), icon: 'i-lucide-check-circle', status: 'RESOLVED' })
  }
  if (props.conversation.status !== 'SNOOZED') {
    items.push({ label: t('conversations.actions.snooze'), icon: 'i-lucide-clock', status: 'SNOOZED' })
  }
  return [[...items]]
})

const statusColor = computed(() => {
  switch (props.conversation.status) {
    case 'OPEN': return 'success' as const
    case 'PENDING': return 'warning' as const
    case 'RESOLVED': return 'info' as const
    case 'SNOOZED': return 'neutral' as const
    default: return 'neutral' as const
  }
})
</script>

<template>
  <UDashboardPanel id="conversations-thread">
    <UDashboardNavbar :title="contactName(conversation)" :toggle="false">
      <template #leading>
        <UButton
          icon="i-lucide-panel-left-close"
          color="neutral"
          variant="ghost"
          class="-ms-1.5"
          @click="emit('close')"
        />
      </template>

      <template #trailing>
        <UBadge
          :label="t(`conversations.filters.${conversation.status.toLowerCase()}`)"
          :color="statusColor"
          variant="subtle"
          size="xs"
        />
      </template>

      <template #right>
        <UDropdownMenu :items="statusItems">
          <UTooltip :text="t('conversations.actions.changeStatus')">
            <UButton
              icon="i-lucide-arrow-up-down"
              color="neutral"
              variant="ghost"
              size="xs"
            />
          </UTooltip>
        </UDropdownMenu>
      </template>
    </UDashboardNavbar>

    <!-- Contact header -->
    <div class="flex flex-col sm:flex-row justify-between gap-1 p-4 sm:px-6 border-b border-default">
      <div class="flex items-start gap-4 sm:my-1.5">
        <UAvatar
          :alt="contactName(conversation)"
          :src="conversation.meta?.sender?.thumbnail ?? undefined"
          size="3xl"
        />
        <div class="min-w-0">
          <p class="font-semibold text-highlighted">
            {{ contactName(conversation) }}
          </p>
          <p class="text-muted text-sm">
            {{ contactIdentifier(conversation) }}
          </p>
          <div v-if="conversation.labels?.length" class="flex items-center gap-1.5 mt-1">
            <span
              v-for="label in conversation.labels"
              :key="label.id"
              class="text-[10px] rounded px-1.5 py-0.5"
              :style="{ backgroundColor: label.color + '20', color: label.color }"
            >
              {{ label.title }}
            </span>
          </div>
        </div>
      </div>
      <div class="max-sm:pl-16 flex flex-col items-end gap-1 text-muted text-sm sm:mt-2">
        <p>{{ conversation.inbox?.name }}</p>
        <p class="text-xs text-dimmed">
          #{{ conversation.displayId }}
        </p>
      </div>
    </div>

    <!-- Messages -->
    <UChatMessages
      class="flex-1"
      :should-scroll-to-bottom="true"
      :should-auto-scroll="true"
      :auto-scroll="false"
      :spacing-offset="80"
    >
      <UChatMessage
        v-for="m in list"
        :id="m.id"
        :key="m.id"
        :role="messageRole(m)"
        :parts="[{ type: 'text', text: m.content ?? '' }]"
        :avatar="messageRole(m) === 'assistant' ? { alt: contactName(conversation), src: conversation.meta?.sender?.thumbnail ?? undefined } : undefined"
        :variant="messageRole(m) === 'system' ? 'naked' : undefined"
      >
        <template v-if="messageRole(m) === 'system'" #content>
          <p class="text-xs text-muted italic text-center w-full">
            {{ m.content }}
          </p>
        </template>
        <template v-else-if="(m.contentAttributes as Record<string, unknown>)?.deleted" #content>
          <p class="italic text-muted">
            {{ t('conversations.message.deleted') }}
          </p>
        </template>
        <template #actions>
          <div class="flex justify-between items-center gap-3">
            <span class="text-[10px] opacity-60">
              {{ format(new Date(m.createdAt), 'HH:mm') }}
            </span>
            <span v-if="m.messageType === 'OUTGOING'" class="text-[10px] opacity-60">
              {{ t(`conversations.message.status.${m.status}`) }}
            </span>
          </div>
        </template>
      </UChatMessage>
    </UChatMessages>

    <!-- Composer -->
    <ConversationComposer :conversation="conversation" />
  </UDashboardPanel>
</template>
