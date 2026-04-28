<script setup lang="ts">
import { useConversationsStore, STATUS_MAP } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'
import { useTeamsStore } from '~/stores/teams'
import { ConfirmModal } from '#components'

const props = defineProps<{
  total?: number
}>()

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
const allSelected = computed(() => (props.total ?? 0) > 0 && selectedIds.value.length === props.total)

function toggleSelectAll(value: boolean | string) {
  if (value) convs.selectAll()
  else convs.clearSelection()
}

async function bulkToggleStatus(status: keyof typeof STATUS_MAP) {
  const accountId = auth.account?.id
  if (!accountId || loading.value) return
  loading.value = true
  try {
    await Promise.all(
      selectedIds.value.map(id =>
        api(`/accounts/${accountId}/conversations/${id}/status`, {
          method: 'PATCH',
          body: { status: STATUS_MAP[status] }
        })
      )
    )
    toast.add({ title: t('conversations.bulk.success', { count: selectedIds.value.length }), icon: 'i-lucide-check-circle', color: 'success' })
    convs.clearSelection()
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
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
    toast.add({ title: t('conversations.bulk.assigned', { count: selectedIds.value.length }), icon: 'i-lucide-check-circle', color: 'success' })
    assignTeamOpen.value = false
    convs.clearSelection()
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
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
    toast.add({ title: t('conversations.bulk.labelsUpdated', { count: selectedIds.value.length }), icon: 'i-lucide-check-circle', color: 'success' })
    labelOpen.value = false
    convs.clearSelection()
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    loading.value = false
  }
}

const route = useRoute()
const router = useRouter()

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
  const ids = [...selectedIds.value]
  try {
    await Promise.all(
      ids.map(id =>
        api(`/accounts/${accountId}/conversations/${id}`, { method: 'DELETE' })
      )
    )
    ids.forEach(id => convs.remove(id))
    // If the open thread was among the deleted ones, drop the URL param so
    // the right pane doesn't sit on a 404 fetch loop.
    const openId = String(route.params.conversationId ?? '')
    if (openId && ids.some(id => String(id) === openId)) {
      router.replace(`/accounts/${accountId}/conversations`)
    }
    toast.add({ title: t('conversations.bulk.deleted', { count: ids.length }), icon: 'i-lucide-check-circle', color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), icon: 'i-lucide-x-circle', color: 'error' })
  } finally {
    loading.value = false
  }
}

// Status actions are split into two groups: terminal transitions (resolve /
// reopen) and snooze durations. Grouping with a separator avoids nesting menus.
const statusMenuItems = computed(() => [
  [
    { label: t('conversations.bulk.resolve'), icon: 'i-lucide-check-circle', onSelect: () => bulkToggleStatus('RESOLVED') },
    { label: t('conversations.bulk.open'), icon: 'i-lucide-message-circle', onSelect: () => bulkToggleStatus('OPEN') }
  ],
  [
    { label: t('conversations.bulk.snooze.1h'), icon: 'i-lucide-clock', onSelect: () => bulkToggleStatus('SNOOZED') },
    { label: t('conversations.bulk.snooze.4h'), icon: 'i-lucide-clock', onSelect: () => bulkToggleStatus('SNOOZED') },
    { label: t('conversations.bulk.snooze.tomorrow'), icon: 'i-lucide-calendar', onSelect: () => bulkToggleStatus('SNOOZED') },
    { label: t('conversations.bulk.snooze.nextWeek'), icon: 'i-lucide-calendar-range', onSelect: () => bulkToggleStatus('SNOOZED') }
  ]
])
</script>

<template>
  <UDashboardToolbar
    v-if="convs.hasSelection"
    class="!min-h-9 !pl-[22px] !pr-2 sm:!pl-[22px] sm:!pr-2"
  >
    <template #left>
      <div class="flex items-center gap-2">
        <UTooltip :text="t('conversations.bulk.selectAll')">
          <UCheckbox
            :model-value="allSelected"
            :aria-label="t('conversations.bulk.selectAll')"
            size="sm"
            @update:model-value="toggleSelectAll"
          />
        </UTooltip>
        <UTooltip :text="t('conversations.bulk.selected', { count: selectedIds.length })">
          <UBadge
            :label="String(selectedIds.length)"
            color="primary"
            variant="subtle"
            size="sm"
          />
        </UTooltip>
      </div>
    </template>

    <template #right>
      <div class="flex items-center gap-1">
        <UDropdownMenu :items="statusMenuItems" :content="{ align: 'start' }">
          <UTooltip :text="t('conversations.bulk.statusLabel')">
            <UButton
              icon="i-lucide-circle-check-big"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('conversations.bulk.statusLabel')"
              :loading="loading"
            />
          </UTooltip>
        </UDropdownMenu>

        <UDropdownMenu :items="[[{ label: t('conversations.bulk.addLabel'), icon: 'i-lucide-tag', onSelect: () => { labelAction = 'add'; labelOpen = true } }, { label: t('conversations.bulk.removeLabel'), icon: 'i-lucide-tag-x', onSelect: () => { labelAction = 'remove'; labelOpen = true } }]]" :content="{ align: 'start' }">
          <UTooltip :text="t('conversations.bulk.labels')">
            <UButton
              icon="i-lucide-tag"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('conversations.bulk.labels')"
            />
          </UTooltip>
        </UDropdownMenu>

        <UDropdownMenu :items="[[{ label: t('conversations.bulk.assignAgent'), icon: 'i-lucide-user-plus', onSelect: () => { assignAgentOpen = true } }, { label: t('conversations.bulk.assignTeam'), icon: 'i-lucide-users', onSelect: () => { assignTeamOpen = true } }]]" :content="{ align: 'start' }">
          <UTooltip :text="t('conversations.bulk.assign')">
            <UButton
              icon="i-lucide-user-plus"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('conversations.bulk.assign')"
            />
          </UTooltip>
        </UDropdownMenu>

        <UTooltip :text="t('conversations.bulk.delete')">
          <UButton
            icon="i-lucide-trash-2"
            color="error"
            variant="ghost"
            size="xs"
            :aria-label="t('conversations.bulk.delete')"
            :loading="loading"
            @click="bulkDelete()"
          />
        </UTooltip>
        <UTooltip :text="t('conversations.bulk.clearSelection')">
          <UButton
            color="neutral"
            variant="ghost"
            size="xs"
            icon="i-lucide-x"
            :aria-label="t('conversations.bulk.clearSelection')"
            @click="convs.clearSelection()"
          />
        </UTooltip>
      </div>
    </template>
  </UDashboardToolbar>

  <template v-if="convs.hasSelection">
    <UModal v-model:open="assignAgentOpen" :title="t('conversations.bulk.assignAgent')">
      <template #body>
        <div class="flex flex-col gap-2">
          <p class="text-sm text-muted">
            {{ t('conversations.bulk.assignAgentDesc', { count: selectedIds.length }) }}
          </p>
          <p class="text-sm text-dimmed">
            {{ t('conversations.bulk.agentsComingSoon') }}
          </p>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="assignTeamOpen" :title="t('conversations.bulk.assignTeam')">
      <template #body>
        <div class="flex flex-col gap-2">
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
      </template>
    </UModal>

    <UModal v-model:open="labelOpen" :title="labelAction === 'add' ? t('conversations.bulk.addLabel') : t('conversations.bulk.removeLabel')">
      <template #body>
        <div class="flex flex-col gap-2">
          <div
            v-for="label in labels.list"
            :key="label.id"
            class="flex items-center gap-2 p-2 rounded hover:bg-elevated cursor-pointer"
            @click="bulkToggleLabel(label.id, labelAction)"
          >
            <span class="w-3 h-3 rounded-full" :style="{ backgroundColor: label.color }" />
            <span class="text-sm">{{ label.title }}</span>
          </div>
        </div>
      </template>
    </UModal>
  </template>
</template>
