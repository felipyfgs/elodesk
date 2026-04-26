<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'
import { useTeamsStore } from '~/stores/teams'
import { useConversationsStore, type Conversation } from '~/stores/conversations'

const props = defineProps<{
  conversationId: string
  currentAssigneeId?: string | null
  currentTeamId?: string | null
}>()

const emit = defineEmits<{
  changed: []
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const teams = useTeamsStore()
const conversations = useConversationsStore()

const loading = ref(false)
const agents = ref<{ id: string, name: string }[]>([])

async function fetchAgents() {
  agents.value = await api<{ id: string, name: string }[]>(`/accounts/${auth.account?.id}/agents`)
}

onMounted(fetchAgents)

const assigneeLabel = computed(() => {
  const agent = agents.value.find(a => a.id === props.currentAssigneeId)
  return agent?.name ?? t('assignment.unassigned')
})

const teamLabel = computed(() => {
  const team = teams.byId(props.currentTeamId ?? '')
  return team?.name ?? t('assignment.unassigned')
})

const agentItems = computed<DropdownMenuItem[][]>(() => [
  [{
    label: t('assignment.unassigned'),
    icon: 'i-lucide-user-minus',
    onSelect: () => assign(null)
  }],
  ...agents.value.map(a => [{
    label: a.name,
    checked: props.currentAssigneeId === a.id,
    onSelect: () => assign(a.id)
  }])
])

const teamItems = computed<DropdownMenuItem[][]>(() => [
  [{
    label: t('assignment.unassigned'),
    icon: 'i-lucide-users',
    onSelect: () => setTeam(null)
  }],
  ...teams.list.map(team => [{
    label: team.name,
    checked: props.currentTeamId === team.id,
    onSelect: () => setTeam(team.id)
  }])
])

// Stores keep IDs as strings, but the backend's BodyParser expects int64 — so
// JSON `"1"` returns 400. Convert to number (or null) before sending in a body.
function toNumericId(v: string | number | null | undefined): number | null {
  if (v === null || v === undefined || v === '') return null
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) ? n : null
}

async function assign(assigneeId: string | null) {
  loading.value = true
  try {
    const conv = await api<Conversation>(`/accounts/${auth.account?.id}/conversations/${props.conversationId}/assignments`, {
      method: 'POST',
      body: { assignee_id: toNumericId(assigneeId), team_id: toNumericId(props.currentTeamId) }
    })
    conversations.upsert(conv)
    emit('changed')
  } finally {
    loading.value = false
  }
}

async function setTeam(teamId: string | null) {
  loading.value = true
  try {
    const conv = await api<Conversation>(`/accounts/${auth.account?.id}/conversations/${props.conversationId}/assignments`, {
      method: 'POST',
      body: { assignee_id: toNumericId(props.currentAssigneeId), team_id: toNumericId(teamId) }
    })
    conversations.upsert(conv)
    emit('changed')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex items-center gap-2">
    <UTooltip :text="t('assignment.assignee')">
      <UDropdownMenu
        :items="agentItems"
        :content="{ align: 'start', collisionPadding: 8 }"
        :disabled="loading"
      >
        <UButton
          :label="assigneeLabel"
          trailing-icon="i-lucide-chevron-down"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="loading"
        />
      </UDropdownMenu>
    </UTooltip>

    <UTooltip :text="t('assignment.team')">
      <UDropdownMenu
        :items="teamItems"
        :content="{ align: 'start', collisionPadding: 8 }"
        :disabled="loading"
      >
        <UButton
          :label="teamLabel"
          trailing-icon="i-lucide-chevron-down"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="loading"
        />
      </UDropdownMenu>
    </UTooltip>
  </div>
</template>
