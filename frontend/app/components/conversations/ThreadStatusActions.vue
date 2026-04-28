<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { STATUS_MAP, type Conversation, type ConversationStatus } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'

type StatusAction = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'

const props = defineProps<{
  conversation: Conversation
}>()

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()
const conversations = useConversationsStore()

const statusLoading = ref(false)

const statusPrimaryLabel = computed(() => (
  props.conversation.status === STATUS_MAP.RESOLVED
    ? t('conversations.actions.open')
    : t('conversations.actions.resolve')
))

// Em <sm o label do botão primário some pra liberar espaço; o ícone
// continua sinalizando a ação (check = resolver, undo = reabrir).
const statusPrimaryIcon = computed(() => (
  props.conversation.status === STATUS_MAP.RESOLVED
    ? 'i-lucide-rotate-ccw'
    : 'i-lucide-check'
))

const statusPrimaryAction = computed<StatusAction>(() => (
  props.conversation.status === STATUS_MAP.RESOLVED ? 'OPEN' : 'RESOLVED'
))

const statusActions: { status: StatusAction, label: string, icon: string }[] = [
  { status: 'OPEN', label: 'conversations.actions.open', icon: 'i-lucide-message-circle' },
  { status: 'PENDING', label: 'conversations.actions.pending', icon: 'i-lucide-clock' },
  { status: 'RESOLVED', label: 'conversations.actions.resolve', icon: 'i-lucide-check-circle' },
  { status: 'SNOOZED', label: 'conversations.actions.snooze', icon: 'i-lucide-bell-off' }
]

const statusItems = computed<DropdownMenuItem[][]>(() => [[
  ...statusActions.map(item => ({
    label: t(item.label),
    icon: item.icon,
    checked: props.conversation.status === STATUS_MAP[item.status],
    onSelect: () => updateStatus(item.status)
  }))
]])

function unwrapConversation(res: Conversation | { payload?: Conversation } | undefined): Conversation | undefined {
  if (!res) return undefined
  if (typeof res === 'object' && 'payload' in res) return res.payload
  return res as Conversation
}

async function updateStatus(status: StatusAction) {
  const accountId = auth.account?.id
  if (!accountId || statusLoading.value) return
  statusLoading.value = true
  try {
    const res = await api<Conversation | { payload?: Conversation }>(`/accounts/${accountId}/conversations/${props.conversation.id}/status`, {
      method: 'PATCH',
      body: { status: STATUS_MAP[status] }
    })
    const conv = unwrapConversation(res)
    conversations.upsert(conv ?? { ...props.conversation, status: STATUS_MAP[status] as ConversationStatus })
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    statusLoading.value = false
  }
}
</script>

<template>
  <UFieldGroup size="sm">
    <!--
      Em <sm o botão fica icon-only pra caber no header. A partir de sm
      mostra o label completo (Resolver/Reabrir). aria-label preserva
      acessibilidade nos dois modos.
    -->
    <UButton
      class="hidden sm:inline-flex"
      :label="statusPrimaryLabel"
      color="neutral"
      variant="soft"
      :loading="statusLoading"
      @click="updateStatus(statusPrimaryAction)"
    />
    <UButton
      class="sm:hidden"
      :icon="statusPrimaryIcon"
      color="neutral"
      variant="soft"
      :loading="statusLoading"
      :aria-label="statusPrimaryLabel"
      @click="updateStatus(statusPrimaryAction)"
    />
    <UDropdownMenu :items="statusItems" :content="{ align: 'end' }">
      <UButton
        icon="i-lucide-chevron-down"
        color="neutral"
        variant="soft"
        :disabled="statusLoading"
        :aria-label="t('conversations.actions.changeStatus')"
      />
    </UDropdownMenu>
  </UFieldGroup>
</template>
