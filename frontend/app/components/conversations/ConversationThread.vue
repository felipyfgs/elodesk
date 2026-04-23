<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore, type Message } from '~/stores/messages'
import { STATUS_MAP, type Conversation, type ConversationStatus } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useAgentsStore } from '~/stores/agents'
import { useTeamsStore } from '~/stores/teams'
import {
  resolveContactName,
  resolveContactIdentifier,
  resolveContactAvatar,
  messageRole,
  messageVariant,
  messageSide,
  messageBubbleKind,
  messageStatusDisplay,
  messageParts,
  messageTime,
  shouldGroupWith,
  hasAttachments,
  getAttachments,
  type BubbleKind
} from '~/utils/chatAdapter'

type StatusAction = 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'

const props = withDefaults(defineProps<{
  conversation: Conversation
  showBack?: boolean
}>(), {
  showBack: false
})

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()
const messages = useMessagesStore()
const conversations = useConversationsStore()
const agents = useAgentsStore()
const teams = useTeamsStore()

const statusLoading = ref(false)
const assignmentLoading = ref(false)
const detailsOpen = ref(true)

const list = computed<Message[]>(() => messages.byConversation[props.conversation.id] ?? [])

async function loadMessages() {
  if (!auth.account?.id) return
  const res = await api<{ payload: Message[] }>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`)
  if (res.payload) {
    messages.set(props.conversation.id, [...res.payload].reverse())
  }
}

watch(() => props.conversation.id, loadMessages, { immediate: true })

onMounted(() => {
  if (!agents.items.length) {
    agents.fetch().catch((err) => {
      console.error('[ConversationThread] failed to fetch agents', err)
    })
  }
})

const contact = computed(() => props.conversation.contactInbox?.contact)
const contactName = computed(() => resolveContactName(props.conversation))
const contactIdentifier = computed(() => resolveContactIdentifier(props.conversation))
const contactAvatar = computed(() => resolveContactAvatar(props.conversation))
const headerSubtitle = computed(() => contactIdentifier.value || props.conversation.inbox?.name || t('conversations.detail.noInbox'))

const statusLabel = computed(() => {
  const keys = ['open', 'resolved', 'pending', 'snoozed']
  return t(`conversations.filters.${keys[props.conversation.status]}`)
})

const statusColor = computed(() => {
  switch (props.conversation.status) {
    case STATUS_MAP.OPEN: return 'success' as const
    case STATUS_MAP.PENDING: return 'warning' as const
    case STATUS_MAP.RESOLVED: return 'info' as const
    case STATUS_MAP.SNOOZED: return 'neutral' as const
    default: return 'neutral' as const
  }
})

const statusPrimaryLabel = computed(() => (
  props.conversation.status === STATUS_MAP.RESOLVED
    ? t('conversations.actions.open')
    : t('conversations.actions.resolve')
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

const currentAssigneeLabel = computed(() => {
  if (props.conversation.meta?.assignee?.name) return props.conversation.meta.assignee.name
  const agent = agents.items.find(a => String(a.userId) === String(props.conversation.assigneeId))
  return agent?.name ?? t('assignment.unassigned')
})

const currentTeamLabel = computed(() => {
  const team = teams.byId(props.conversation.teamId ?? '')
  return team?.name ?? t('assignment.unassigned')
})

const agentItems = computed<DropdownMenuItem[][]>(() => [
  [{
    label: t('assignment.unassigned'),
    icon: 'i-lucide-user-minus',
    checked: !props.conversation.assigneeId,
    onSelect: () => updateAssignment(null, props.conversation.teamId ?? null)
  }],
  ...agents.items.map(agent => [{
    label: agent.name || agent.email,
    checked: String(agent.userId) === String(props.conversation.assigneeId),
    onSelect: () => updateAssignment(String(agent.userId), props.conversation.teamId ?? null)
  }])
])

const teamItems = computed<DropdownMenuItem[][]>(() => [
  [{
    label: t('assignment.unassigned'),
    icon: 'i-lucide-users',
    checked: !props.conversation.teamId,
    onSelect: () => updateAssignment(props.conversation.assigneeId ?? null, null)
  }],
  ...teams.list.map(team => [{
    label: team.name,
    checked: String(team.id) === String(props.conversation.teamId),
    onSelect: () => updateAssignment(props.conversation.assigneeId ?? null, team.id)
  }])
])

const priorityItems = computed<DropdownMenuItem[][]>(() => [[
  { label: t('conversations.detail.none'), checked: true }
]])

const detailSections = computed(() => [
  { value: 'actions', label: t('conversations.detail.actions'), icon: 'i-lucide-bolt' },
  { value: 'macros', label: t('conversations.detail.macros'), icon: 'i-lucide-command' },
  { value: 'conversationInfo', label: t('conversations.detail.conversationInfo'), icon: 'i-lucide-info' },
  { value: 'contactAttributes', label: t('conversations.detail.contactAttributes'), icon: 'i-lucide-tags' },
  { value: 'contactNotes', label: t('conversations.detail.contactNotes'), icon: 'i-lucide-notebook-pen' },
  { value: 'previousConversations', label: t('conversations.detail.previousConversations'), icon: 'i-lucide-history' },
  { value: 'participants', label: t('conversations.detail.participants'), icon: 'i-lucide-users' },
  { value: 'linkedIssues', label: t('conversations.detail.linkedIssues'), icon: 'i-lucide-git-pull-request' }
])

const contactRows = computed(() => [
  { icon: 'i-lucide-hash', title: t('conversations.detail.conversation'), value: `#${props.conversation.displayId}` },
  { icon: 'i-lucide-at-sign', title: t('conversations.detail.identifier'), value: contact.value?.waJid || contact.value?.email || contactIdentifier.value || t('conversations.detail.noIdentifier') },
  { icon: 'i-lucide-phone', title: t('conversations.detail.phone'), value: contact.value?.phoneNumber },
  { icon: 'i-lucide-inbox', title: t('conversations.detail.inbox'), value: props.conversation.inbox?.name },
  { icon: 'i-lucide-radio', title: t('conversations.detail.channel'), value: channelBadge.value }
].filter(row => row.value))

const channelBadge = computed(() => {
  const type = props.conversation.inbox?.channelType || props.conversation.meta?.channel || ''
  if (type.includes('Whatsapp') || type.includes('Twilio')) return 'WP'
  if (type.includes('Sms')) return 'SMS'
  if (type.includes('Email')) return 'EM'
  if (type.includes('Instagram')) return 'IG'
  if (type.includes('Facebook')) return 'FB'
  if (type.includes('Telegram')) return 'TG'
  return 'CH'
})

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
      body: { status }
    })
    const conv = unwrapConversation(res)
    conversations.upsert(conv ?? { ...props.conversation, status: STATUS_MAP[status] as ConversationStatus })
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    statusLoading.value = false
  }
}

async function updateAssignment(assigneeId: string | null, teamId: string | null) {
  const accountId = auth.account?.id
  if (!accountId || assignmentLoading.value) return
  assignmentLoading.value = true
  try {
    const res = await api<Conversation | { payload?: Conversation }>(`/accounts/${accountId}/conversations/${props.conversation.id}/assignments`, {
      method: 'POST',
      body: { assignee_id: assigneeId, team_id: teamId }
    })
    const conv = unwrapConversation(res)
    if (conv) {
      conversations.upsert(conv)
    } else {
      conversations.upsert({ ...props.conversation, assigneeId, teamId })
    }
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    assignmentLoading.value = false
  }
}

function assignToMe() {
  if (!auth.user?.id) return
  updateAssignment(auth.user.id, props.conversation.teamId ?? null)
}

function isGrouped(index: number): boolean {
  if (index === 0) return false
  const prev = list.value[index - 1]
  const curr = list.value[index]
  if (!prev || !curr) return false
  return shouldGroupWith(prev, curr)
}

function bubbleKind(m: Message): BubbleKind {
  return messageBubbleKind(m)
}

function messageUi(m: Message) {
  const kind = bubbleKind(m)
  if (kind === 'activity' || kind === 'template') {
    return {
      root: 'flex justify-center',
      container: 'justify-center pb-2',
      content: '!w-fit min-h-0 rounded-lg bg-elevated px-3 py-1 text-xs text-muted',
      actions: 'hidden'
    }
  }

  const outgoing = messageSide(m) === 'right'
  return {
    container: outgoing ? 'justify-end pb-4' : 'justify-start pb-4',
    content: [
      'max-w-[34rem] whitespace-pre-wrap break-words px-3.5 py-2 text-sm shadow-sm ring-0',
      outgoing
        ? 'rounded-lg rounded-br-sm bg-primary text-inverted'
        : 'rounded-lg rounded-bl-sm bg-elevated text-highlighted'
    ],
    actions: outgoing ? 'right-1 text-dimmed' : 'left-1 text-dimmed'
  }
}
</script>

<template>
  <UDashboardPanel id="conversations-thread" class="min-w-0 flex-1">
    <div class="flex min-h-0 flex-1 bg-default">
      <section class="flex min-w-0 flex-1 flex-col bg-default">
        <header class="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-default px-3 sm:px-4">
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <UTooltip v-if="showBack" :text="t('common.close')">
              <UButton
                icon="i-lucide-arrow-left"
                color="neutral"
                variant="ghost"
                size="sm"
                :aria-label="t('common.close')"
                @click="emit('close')"
              />
            </UTooltip>

            <div class="-ms-1 flex min-w-0 flex-1 items-center gap-3 px-2 py-1.5">
              <UAvatar
                :alt="contactName"
                :src="contactAvatar"
                size="md"
                class="shrink-0"
              />
              <div class="min-w-0 flex-1">
                <div class="flex min-w-0 items-center gap-2">
                  <p class="truncate text-sm font-semibold text-highlighted">
                    {{ contactName }}
                  </p>
                  <span class="hidden shrink-0 text-xs font-medium text-muted sm:inline">
                    #{{ conversation.displayId }}
                  </span>
                </div>
                <div class="mt-0.5 flex min-w-0 items-center gap-1.5">
                  <span class="truncate text-xs text-muted">{{ headerSubtitle }}</span>
                  <UBadge
                    :label="statusLabel"
                    :color="statusColor"
                    variant="subtle"
                    size="xs"
                    class="shrink-0"
                  />
                </div>
              </div>
            </div>
          </div>

          <div class="flex shrink-0 items-center gap-1">
            <UDropdownMenu :items="statusItems" :content="{ align: 'end' }">
              <UButton
                :label="statusPrimaryLabel"
                trailing-icon="i-lucide-chevron-down"
                color="neutral"
                variant="soft"
                size="sm"
                :loading="statusLoading"
              />
            </UDropdownMenu>
            <UTooltip :text="t('conversations.detail.contacts')">
              <UButton
                icon="i-lucide-user-round"
                color="neutral"
                :variant="detailsOpen ? 'soft' : 'ghost'"
                size="xs"
                class="hidden lg:inline-flex"
                :aria-label="t('conversations.detail.contacts')"
                @click="detailsOpen = !detailsOpen"
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

        <div class="flex min-h-0 flex-1 flex-col bg-default">
          <div class="min-h-0 flex-1 overflow-y-auto px-3 sm:px-4">
            <UChatMessages
              class="mx-auto w-full max-w-3xl px-0 py-4"
              :should-scroll-to-bottom="true"
              :auto-scroll="true"
              :spacing-offset="160"
            >
              <div v-if="!list.length" class="flex flex-col items-center justify-center py-12 text-muted">
                <UIcon name="i-lucide-message-circle-off" class="mb-2 size-8 text-dimmed" />
                <p class="text-sm">
                  {{ t('conversations.thread.empty') }}
                </p>
              </div>

              <UChatMessage
                v-for="(m, i) in list"
                :id="String(m.id)"
                :key="m.id"
                :role="messageRole(m)"
                :variant="messageVariant(m)"
                :side="messageSide(m)"
                :parts="messageParts(m)"
                :compact="isGrouped(i)"
                :ui="messageUi(m)"
              >
                <template v-if="bubbleKind(m) === 'deleted'" #content>
                  <p class="font-semibold">
                    {{ t('conversations.message.deleted') }}
                  </p>
                </template>

                <template v-else-if="bubbleKind(m) === 'private'" #content>
                  <div class="mb-1 flex items-center gap-1.5">
                    <UIcon name="i-lucide-lock" class="size-3 text-warning" />
                    <span class="text-[10px] font-medium text-warning">{{ t('conversations.message.private') }}</span>
                  </div>
                  <p>{{ m.content }}</p>
                </template>

                <template v-else-if="bubbleKind(m) === 'activity'" #content>
                  <p>
                    {{ m.content }}
                  </p>
                </template>

                <template v-else-if="bubbleKind(m) === 'error'" #content>
                  <p class="text-error">
                    {{ m.content }}
                  </p>
                </template>

                <template v-if="hasAttachments(m)" #files>
                  <div class="mt-1 flex flex-wrap gap-2">
                    <a
                      v-for="(att, ai) in getAttachments(m)"
                      :key="ai"
                      :href="att.fileUrl"
                      target="_blank"
                      class="inline-flex items-center gap-1.5 rounded-md bg-default px-2 py-1 text-xs text-primary ring ring-default hover:underline"
                    >
                      <UIcon name="i-lucide-paperclip" class="size-3" />
                      {{ att.fileType || 'attachment' }}
                    </a>
                  </div>
                </template>

                <template #actions>
                  <div class="flex items-center gap-1.5 text-[10px] text-muted">
                    <span>{{ messageTime(m) }}</span>
                    <UIcon
                      v-if="m.messageType === 1"
                      :name="messageStatusDisplay(m, t).icon"
                      :class="['size-3', messageStatusDisplay(m, t).color]"
                    />
                  </div>
                </template>
              </UChatMessage>
            </UChatMessages>
          </div>

          <ConversationsConversationComposer :conversation="conversation" />
        </div>
      </section>

      <aside
        v-if="detailsOpen"
        class="hidden w-72 shrink-0 flex-col border-l border-default bg-default lg:flex xl:w-80"
      >
        <div class="flex h-14 shrink-0 items-center justify-between border-b border-default px-4">
          <h2 class="text-sm font-semibold text-highlighted">
            {{ t('conversations.detail.contacts') }}
          </h2>
          <UButton
            icon="i-lucide-x"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('common.close')"
            @click="detailsOpen = false"
          />
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto">
          <section class="border-b border-default px-4 py-4">
            <div class="flex items-start gap-3">
              <div class="relative shrink-0">
                <UAvatar
                  :alt="contactName"
                  :src="contactAvatar"
                  size="xl"
                />
                <span class="absolute -bottom-1 -right-1 rounded-md bg-elevated px-1 py-0.5 text-[10px] font-semibold text-muted ring ring-default">
                  {{ channelBadge }}
                </span>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-start gap-1.5">
                  <p class="truncate text-base font-semibold text-highlighted">
                    {{ contactName }}
                  </p>
                  <UBadge
                    :label="statusLabel"
                    :color="statusColor"
                    variant="subtle"
                    size="xs"
                    class="mt-0.5 shrink-0"
                  />
                </div>
                <p class="mt-0.5 truncate text-sm text-muted">
                  {{ contactIdentifier || t('conversations.detail.noIdentifier') }}
                </p>
                <div class="mt-2 flex flex-wrap gap-1.5">
                  <UBadge
                    :label="`#${conversation.displayId}`"
                    color="neutral"
                    variant="soft"
                    size="xs"
                  />
                  <UBadge
                    v-if="conversation.inbox?.name"
                    :label="conversation.inbox.name"
                    color="neutral"
                    variant="soft"
                    size="xs"
                    class="max-w-full truncate"
                  />
                </div>
              </div>
            </div>

            <div class="mt-4 grid grid-cols-4 gap-2">
              <UTooltip :text="t('conversations.detail.message')">
                <UButton
                  icon="i-lucide-message-circle"
                  color="neutral"
                  variant="soft"
                  size="sm"
                  block
                  :aria-label="t('conversations.detail.message')"
                />
              </UTooltip>
              <UTooltip :text="t('common.edit')">
                <UButton
                  icon="i-lucide-pencil"
                  color="neutral"
                  variant="soft"
                  size="sm"
                  block
                  :aria-label="t('common.edit')"
                />
              </UTooltip>
              <UTooltip :text="t('conversations.compose.voice')">
                <UButton
                  icon="i-lucide-mic"
                  color="neutral"
                  variant="soft"
                  size="sm"
                  block
                  :aria-label="t('conversations.compose.voice')"
                />
              </UTooltip>
              <UTooltip :text="t('common.delete')">
                <UButton
                  icon="i-lucide-trash-2"
                  color="error"
                  variant="soft"
                  size="sm"
                  block
                  :aria-label="t('common.delete')"
                />
              </UTooltip>
            </div>

            <dl class="mt-4 space-y-3">
              <div
                v-for="row in contactRows"
                :key="`${row.icon}-${row.title}`"
                class="grid min-w-0 grid-cols-[1rem_minmax(0,1fr)] gap-2"
              >
                <UIcon :name="row.icon" class="mt-0.5 size-4 shrink-0 text-muted" />
                <div class="min-w-0">
                  <dt class="text-[11px] font-medium uppercase text-dimmed">
                    {{ row.title }}
                  </dt>
                  <dd class="truncate text-sm text-muted">
                    {{ row.value }}
                  </dd>
                </div>
              </div>
            </dl>
          </section>

          <UAccordion
            :items="detailSections"
            type="multiple"
            :default-value="['actions']"
            :ui="{
              root: 'space-y-2 p-2',
              item: 'overflow-hidden rounded-md border border-default bg-default last:border-b',
              trigger: 'px-3 py-3 text-sm hover:bg-elevated/60',
              leadingIcon: 'size-4 text-muted',
              body: 'px-3 pb-3 text-sm'
            }"
          >
            <template #trailing="{ open }">
              <UIcon
                name="i-lucide-plus"
                :class="['ms-auto size-4 shrink-0 text-muted transition-transform', open ? 'rotate-45' : '']"
              />
            </template>

            <template #body="{ item }">
              <div v-if="item.value === 'actions'" class="space-y-3">
                <div class="flex items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold text-highlighted">
                    {{ t('conversations.detail.assignedAgent') }}
                  </h3>
                  <UButton
                    :label="t('conversations.detail.assignToMe')"
                    icon="i-lucide-arrow-right"
                    color="primary"
                    variant="link"
                    size="xs"
                    :loading="assignmentLoading"
                    @click="assignToMe"
                  />
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.assignedAgent') }}
                  </p>
                  <UDropdownMenu :items="agentItems" :content="{ align: 'start' }" :disabled="assignmentLoading">
                    <UButton
                      :label="currentAssigneeLabel"
                      trailing-icon="i-lucide-chevron-down"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      block
                      :loading="assignmentLoading"
                      class="justify-between"
                    />
                  </UDropdownMenu>
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.assignedTeam') }}
                  </p>
                  <UDropdownMenu :items="teamItems" :content="{ align: 'start' }" :disabled="assignmentLoading">
                    <UButton
                      :label="currentTeamLabel"
                      trailing-icon="i-lucide-chevron-down"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      block
                      :loading="assignmentLoading"
                      class="justify-between"
                    />
                  </UDropdownMenu>
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.priority') }}
                  </p>
                  <UDropdownMenu :items="priorityItems" :content="{ align: 'start' }">
                    <UButton
                      :label="t('conversations.detail.none')"
                      trailing-icon="i-lucide-chevron-down"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      block
                      class="justify-between"
                    />
                  </UDropdownMenu>
                </div>

                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted">
                    {{ t('conversations.detail.labels') }}
                  </p>
                  <div v-if="conversation.labels?.length" class="flex flex-wrap gap-1.5">
                    <span
                      v-for="label in conversation.labels"
                      :key="label.id"
                      class="rounded-md px-2 py-1 text-xs"
                      :style="{ backgroundColor: `${label.color}20`, color: label.color }"
                    >
                      {{ label.title }}
                    </span>
                  </div>
                  <UButton
                    v-else
                    :label="t('conversations.detail.addLabels')"
                    icon="i-lucide-plus"
                    color="primary"
                    variant="soft"
                    size="xs"
                  />
                </div>
              </div>

              <div v-else class="flex items-center gap-2 rounded-md bg-elevated/50 px-3 py-2 text-sm text-muted">
                <UIcon name="i-lucide-circle-dashed" class="size-4 shrink-0 text-dimmed" />
                {{ t('conversations.detail.emptySection') }}
              </div>
            </template>
          </UAccordion>
        </div>
      </aside>
    </div>
  </UDashboardPanel>
</template>
