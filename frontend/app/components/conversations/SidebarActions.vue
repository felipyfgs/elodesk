<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { Conversation } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useAgentsStore } from '~/stores/agents'
import { useTeamsStore } from '~/stores/teams'
import { useAuthStore } from '~/stores/auth'

const props = defineProps<{
  conversation: Conversation
}>()

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()
const conversations = useConversationsStore()
const agents = useAgentsStore()
const teams = useTeamsStore()

const assignmentLoading = ref(false)

// Stores keep IDs as strings, but the backend's BodyParser expects int64 — so
// JSON `"1"` returns 400. Convert to number (or null) before sending in a body.
function toNumericId(v: string | number | null | undefined): number | null {
  if (v === null || v === undefined || v === '') return null
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) ? n : null
}

function unwrapConversation(res: Conversation | { payload?: Conversation } | undefined): Conversation | undefined {
  if (!res) return undefined
  if (typeof res === 'object' && 'payload' in res) return res.payload
  return res as Conversation
}

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

async function updateAssignment(assigneeId: string | null, teamId: string | null) {
  const accountId = auth.account?.id
  if (!accountId || assignmentLoading.value) return
  assignmentLoading.value = true
  try {
    const res = await api<Conversation | { payload?: Conversation }>(`/accounts/${accountId}/conversations/${props.conversation.id}/assignments`, {
      method: 'POST',
      body: { assignee_id: toNumericId(assigneeId), team_id: toNumericId(teamId) }
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
</script>

<template>
  <div class="space-y-3">
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
          :key="label"
          class="rounded-md bg-elevated px-2 py-1 text-xs text-muted"
        >
          {{ label }}
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
</template>
