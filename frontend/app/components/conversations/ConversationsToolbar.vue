<script setup lang="ts">
import { useConversationsStore } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'
import { useTeamsStore } from '~/stores/teams'
import { ConfirmModal } from '#components'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const convs = useConversationsStore()
const auth = useAuthStore()
const labels = useLabelsStore()
const teams = useTeamsStore()

const confirm = useOverlay().create(ConfirmModal)

const assignAgentOpen = ref(false)
const assignTeamOpen = ref(false)
const labelOpen = ref(false)
const labelAction = ref<'add' | 'remove'>('add')
const loading = ref(false)

const selectedIds = computed(() => convs.selection)

const snoozeItems = computed(() => [
  [{
    label: t('conversations.bulk.snooze.1h'),
    icon: 'i-lucide-clock',
    click: () => snooze('1h')
  }, {
    label: t('conversations.bulk.snooze.4h'),
    icon: 'i-lucide-clock',
    click: () => snooze('4h')
  }, {
    label: t('conversations.bulk.snooze.tomorrow'),
    icon: 'i-lucide-calendar',
    click: () => snooze('tomorrow')
  }, {
    label: t('conversations.bulk.snooze.nextWeek'),
    icon: 'i-lucide-calendar-range',
    click: () => snooze('next_week')
  }]
])

function snooze(duration: string) {
  const now = new Date()
  let until: Date
  switch (duration) {
    case '1h':
      until = new Date(now.getTime() + 60 * 60 * 1000)
      break
    case '4h':
      until = new Date(now.getTime() + 4 * 60 * 60 * 1000)
      break
    case 'tomorrow':
      until = new Date(now)
      until.setDate(until.getDate() + 1)
      until.setHours(9, 0, 0, 0)
      break
    case 'next_week':
      until = new Date(now)
      until.setDate(until.getDate() + 7)
      until.setHours(9, 0, 0, 0)
      break
    default:
      until = new Date(now.getTime() + 60 * 60 * 1000)
  }
  bulkToggleStatus('SNOOZED')
}

async function bulkToggleStatus(status: string) {
  const accountId = auth.account?.id
  if (!accountId || loading.value) return
  loading.value = true
  try {
    await Promise.all(
      selectedIds.value.map(id =>
        api(`/accounts/${accountId}/conversations/${id}/status`, {
          method: 'PATCH',
          body: { status }
        })
      )
    )
    toast.add({
      title: t('conversations.bulk.success', { count: selectedIds.value.length }),
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    convs.clearSelection()
  } catch {
    toast.add({
      title: t('common.error'),
      icon: 'i-lucide-x-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

async function _bulkAssignAgent(userId: string) {
  const accountId = auth.account?.id
  if (!accountId || loading.value) return
  loading.value = true
  try {
    await Promise.all(
      selectedIds.value.map(id =>
        api(`/accounts/${accountId}/conversations/${id}/assignments`, {
          method: 'POST',
          body: { assignee_id: Number(userId) }
        })
      )
    )
    toast.add({
      title: t('conversations.bulk.assigned', { count: selectedIds.value.length }),
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    assignAgentOpen.value = false
    convs.clearSelection()
  } catch {
    toast.add({
      title: t('common.error'),
      icon: 'i-lucide-x-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

async function bulkAssignTeam(teamId: string) {
  const accountId = auth.account?.id
  if (!accountId || loading.value) return
  loading.value = true
  try {
    await Promise.all(
      selectedIds.value.map(id =>
        api(`/accounts/${accountId}/conversations/${id}/assignments`, {
          method: 'POST',
          body: { team_id: Number(teamId) }
        })
      )
    )
    toast.add({
      title: t('conversations.bulk.assigned', { count: selectedIds.value.length }),
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    assignTeamOpen.value = false
    convs.clearSelection()
  } catch {
    toast.add({
      title: t('common.error'),
      icon: 'i-lucide-x-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

async function bulkToggleLabel(labelId: string, action: 'add' | 'remove') {
  const accountId = auth.account?.id
  if (!accountId || loading.value) return
  loading.value = true
  try {
    await Promise.all(
      selectedIds.value.map(id =>
        api(`/accounts/${accountId}/conversations/${id}/labels`, {
          method: action === 'add' ? 'POST' : 'DELETE',
          body: action === 'add' ? { label_ids: [Number(labelId)] } : undefined,
          params: action === 'remove' ? { label_id: labelId } : undefined
        })
      )
    )
    toast.add({
      title: t('conversations.bulk.labelsUpdated', { count: selectedIds.value.length }),
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    labelOpen.value = false
    convs.clearSelection()
  } catch {
    toast.add({
      title: t('common.error'),
      icon: 'i-lucide-x-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

async function bulkDelete() {
  const accountId = auth.account?.id
  if (!accountId || loading.value) return

  const confirmed = await confirm.open({
    title: t('conversations.bulk.deleteConfirm'),
    description: t('conversations.bulk.deleteDesc', { count: selectedIds.value.length }),
    confirmColor: 'error'
  }).result

  if (!confirmed) return

  loading.value = true
  try {
    // Resolve conversations (soft delete not available, resolve as alternative)
    await Promise.all(
      selectedIds.value.map(id =>
        api(`/accounts/${accountId}/conversations/${id}/status`, {
          method: 'PATCH',
          body: { status: 'RESOLVED' }
        })
      )
    )
    toast.add({
      title: t('conversations.bulk.resolved', { count: selectedIds.value.length }),
      icon: 'i-lucide-check-circle',
      color: 'success'
    })
    convs.clearSelection()
  } catch {
    toast.add({
      title: t('common.error'),
      icon: 'i-lucide-x-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UDashboardToolbar v-if="convs.hasSelection">
    <template #leading>
      <span class="text-sm font-medium">
        {{ t('conversations.bulk.selected', { count: selectedIds.length }) }}
      </span>
    </template>

    <template #trailing>
      <div class="flex items-center gap-1">
        <UButton
          :label="t('conversations.bulk.resolve')"
          icon="i-lucide-check-circle"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="loading"
          @click="bulkToggleStatus('RESOLVED')"
        />

        <UDropdownMenu :items="snoozeItems">
          <UButton
            :label="t('conversations.bulk.snooze.label')"
            icon="i-lucide-clock"
            color="neutral"
            variant="ghost"
            size="xs"
          />
        </UDropdownMenu>

        <UButton
          :label="t('conversations.bulk.open')"
          icon="i-lucide-message-circle"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="loading"
          @click="bulkToggleStatus('OPEN')"
        />

        <UDropdownMenu
          :items="[[{
            label: t('conversations.bulk.addLabel'),
            icon: 'i-lucide-tag',
            click: () => { labelAction = 'add'; labelOpen = true }
          }, {
            label: t('conversations.bulk.removeLabel'),
            icon: 'i-lucide-tag-x',
            click: () => { labelAction = 'remove'; labelOpen = true }
          }]]"
        >
          <UButton
            :label="t('conversations.bulk.labels')"
            icon="i-lucide-tag"
            color="neutral"
            variant="ghost"
            size="xs"
          />
        </UDropdownMenu>

        <UDropdownMenu
          :items="[[{
            label: t('conversations.bulk.assignAgent'),
            icon: 'i-lucide-user-plus',
            click: () => { assignAgentOpen = true }
          }, {
            label: t('conversations.bulk.assignTeam'),
            icon: 'i-lucide-users',
            click: () => { assignTeamOpen = true }
          }]]"
        >
          <UButton
            :label="t('conversations.bulk.assign')"
            icon="i-lucide-user-plus"
            color="neutral"
            variant="ghost"
            size="xs"
          />
        </UDropdownMenu>

        <UButton
          :label="t('conversations.bulk.delete')"
          icon="i-lucide-trash-2"
          color="error"
          variant="ghost"
          size="xs"
          :loading="loading"
          @click="bulkDelete()"
        />

        <UButton
          :label="t('conversations.bulk.clearSelection')"
          color="neutral"
          variant="ghost"
          size="xs"
          icon="i-lucide-x"
          @click="convs.clearSelection()"
        />
      </div>
    </template>
  </UDashboardToolbar>

  <!-- Assign Agent Modal -->
  <UModal v-model:open="assignAgentOpen" :title="t('conversations.bulk.assignAgent')">
    <div class="p-4 flex flex-col gap-2">
      <p class="text-sm text-muted">
        {{ t('conversations.bulk.assignAgentDesc', { count: selectedIds.length }) }}
      </p>
      <!-- Agent selection would be populated from agents endpoint (Fase 4) -->
      <p class="text-sm text-dimmed">
        {{ t('conversations.bulk.agentsComingSoon') }}
      </p>
    </div>
  </UModal>

  <!-- Assign Team Modal -->
  <UModal v-model:open="assignTeamOpen" :title="t('conversations.bulk.assignTeam')">
    <div class="p-4 flex flex-col gap-2">
      <p class="text-sm text-muted">
        {{ t('conversations.bulk.assignTeamDesc', { count: selectedIds.length }) }}
      </p>
      <div
        v-for="team in teams.list"
        :key="team.id"
        class="flex items-center gap-2 p-2 rounded hover:bg-elevated cursor-pointer"
        @click="bulkAssignTeam(team.id)"
      >
        <UIcon name="i-lucide-users" class="size-4 text-muted" />
        <span class="text-sm">{{ team.name }}</span>
      </div>
    </div>
  </UModal>

  <!-- Label Modal -->
  <UModal v-model:open="labelOpen" :title="labelAction === 'add' ? t('conversations.bulk.addLabel') : t('conversations.bulk.removeLabel')">
    <div class="p-4 flex flex-col gap-2">
      <div
        v-for="label in labels.list"
        :key="label.id"
        :label="label.title"
        class="flex items-center gap-2 p-2 rounded hover:bg-elevated cursor-pointer"
        @click="bulkToggleLabel(label.id, labelAction)"
      >
        <span class="w-3 h-3 rounded-full" :style="{ backgroundColor: label.color }" />
        <span class="text-sm">{{ label.title }}</span>
      </div>
    </div>
  </UModal>
</template>
