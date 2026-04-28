<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'
import { useConversationsStore } from '~/stores/conversations'
import { useInboxesStore } from '~/stores/inboxes'
import { useLabelsStore } from '~/stores/labels'
import { useTeamsStore } from '~/stores/teams'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
import { channelIcon } from '~/utils/channels'

const props = defineProps<{
  currentStatus: { label: string, icon: string }
  currentSort: { icon: string }
  statusMenuItems: DropdownMenuItem[]
  sortMenuItems: DropdownMenuItem[]
  advancedQuery: FilterQueryPayload | null
  activeSavedFilter: SavedFilter | null
}>()

const emit = defineEmits<{
  openAdvancedFilter: []
  applySavedFilter: [filter: SavedFilter]
}>()

const { t } = useI18n()
const convs = useConversationsStore()
const inboxes = useInboxesStore()
const labels = useLabelsStore()
const teams = useTeamsStore()
const savedFilters = useSavedFiltersStore()

const disabled = computed(() => !!props.advancedQuery)

// Scope dropdowns mutate convs.filters directly (popover-driven filtering, see
// middleware/conversations-scope.ts). Each picker is multi-select except saved
// filters, which apply a stored advancedQuery via emit.
function toggle(field: 'inboxIds' | 'labelIds' | 'teamIds', id: string) {
  const current = (convs.filters[field] ?? []).map(String)
  const next = current.includes(id) ? current.filter(v => v !== id) : [...current, id]
  convs.setFilters({ [field]: next.length ? next : undefined })
}

function clearScope(field: 'inboxIds' | 'labelIds' | 'teamIds') {
  convs.setFilters({ [field]: undefined })
}

const inboxItems = computed<DropdownMenuItem[][]>(() => {
  const selected = new Set((convs.filters.inboxIds ?? []).map(String))
  const items: DropdownMenuItem[] = inboxes.list.map(ix => ({
    label: ix.name,
    icon: channelIcon(ix.channelType),
    type: 'checkbox' as const,
    checked: selected.has(String(ix.id)),
    onSelect: () => toggle('inboxIds', String(ix.id))
  }))
  if (!items.length) {
    items.push({ label: t('conversations.empty'), disabled: true })
  }
  const footer: DropdownMenuItem[] = selected.size
    ? [{ label: t('conversations.scope.clear'), icon: 'i-lucide-x', onSelect: () => clearScope('inboxIds') }]
    : []
  return footer.length ? [items, footer] : [items]
})

const labelItems = computed<DropdownMenuItem[][]>(() => {
  const selected = new Set(convs.filters.labelIds ?? [])
  const items: DropdownMenuItem[] = labels.list.map(l => ({
    label: l.title,
    type: 'checkbox' as const,
    checked: selected.has(l.title),
    onSelect: () => toggle('labelIds', l.title)
  }))
  if (!items.length) items.push({ label: t('conversations.empty'), disabled: true })
  const footer: DropdownMenuItem[] = selected.size
    ? [{ label: t('conversations.scope.clear'), icon: 'i-lucide-x', onSelect: () => clearScope('labelIds') }]
    : []
  return footer.length ? [items, footer] : [items]
})

const teamItems = computed<DropdownMenuItem[][]>(() => {
  const selected = new Set((convs.filters.teamIds ?? []).map(String))
  const items: DropdownMenuItem[] = teams.list.map(tm => ({
    label: tm.name,
    icon: 'i-lucide-users-round',
    type: 'checkbox' as const,
    checked: selected.has(String(tm.id)),
    onSelect: () => toggle('teamIds', String(tm.id))
  }))
  if (!items.length) items.push({ label: t('conversations.empty'), disabled: true })
  const footer: DropdownMenuItem[] = selected.size
    ? [{ label: t('conversations.scope.clear'), icon: 'i-lucide-x', onSelect: () => clearScope('teamIds') }]
    : []
  return footer.length ? [items, footer] : [items]
})

const savedFilterItems = computed<DropdownMenuItem[]>(() => {
  const list = savedFilters.conversationFilters
  if (!list.length) return [{ label: t('savedFilters.empty'), disabled: true }]
  return list.map(f => ({
    label: f.name,
    icon: 'i-lucide-filter',
    type: 'checkbox' as const,
    checked: props.activeSavedFilter?.id === f.id,
    onSelect: () => emit('applySavedFilter', f)
  }))
})

const inboxBadge = computed(() => convs.filters.inboxIds?.length ?? 0)
const labelBadge = computed(() => convs.filters.labelIds?.length ?? 0)
const teamBadge = computed(() => convs.filters.teamIds?.length ?? 0)

// Count of orthogonal flags layered on top of status. Today: unattended,
// unassigned. Future ones (mentions/high-priority/etc) plug in here.
const statusFlagCount = computed(() => {
  let n = 0
  if (convs.filters.conversationType === 'unattended') n++
  if (convs.filters.unassignedOnly) n++
  return n
})
</script>

<template>
  <div class="flex items-center justify-between gap-1 border-b border-default px-2 py-1">
    <div class="flex items-center gap-1 min-w-0 overflow-x-auto">
      <UDropdownMenu :items="statusMenuItems" :content="{ align: 'start' }">
        <UButton
          :label="currentStatus.label"
          :icon="currentStatus.icon"
          color="neutral"
          variant="ghost"
          size="sm"
          :disabled="disabled"
        >
          <template #trailing>
            <UBadge v-if="statusFlagCount" :label="`+${statusFlagCount}`" size="sm" color="primary" variant="subtle" />
            <UIcon name="i-lucide-chevrons-up-down" class="size-3.5 text-muted" />
          </template>
        </UButton>
      </UDropdownMenu>

      <UDropdownMenu :items="inboxItems" :content="{ align: 'start' }">
        <UTooltip :text="t('nav.channels')">
          <UButton
            icon="i-lucide-inbox"
            :aria-label="t('nav.channels')"
            :color="inboxBadge ? 'primary' : 'neutral'"
            :variant="inboxBadge ? 'soft' : 'ghost'"
            size="xs"
            :disabled="disabled"
          >
            <UBadge v-if="inboxBadge" :label="inboxBadge" size="sm" color="primary" variant="subtle" />
          </UButton>
        </UTooltip>
      </UDropdownMenu>

      <UDropdownMenu :items="labelItems" :content="{ align: 'start' }">
        <UTooltip :text="t('nav.labels')">
          <UButton
            icon="i-lucide-tag"
            :aria-label="t('nav.labels')"
            :color="labelBadge ? 'primary' : 'neutral'"
            :variant="labelBadge ? 'soft' : 'ghost'"
            size="xs"
            :disabled="disabled"
          >
            <UBadge v-if="labelBadge" :label="labelBadge" size="sm" color="primary" variant="subtle" />
          </UButton>
        </UTooltip>
      </UDropdownMenu>

      <UDropdownMenu v-if="teams.list.length" :items="teamItems" :content="{ align: 'start' }">
        <UTooltip :text="t('nav.teams')">
          <UButton
            icon="i-lucide-users-round"
            :aria-label="t('nav.teams')"
            :color="teamBadge ? 'primary' : 'neutral'"
            :variant="teamBadge ? 'soft' : 'ghost'"
            size="xs"
            :disabled="disabled"
          >
            <UBadge v-if="teamBadge" :label="teamBadge" size="sm" color="primary" variant="subtle" />
          </UButton>
        </UTooltip>
      </UDropdownMenu>

      <UDropdownMenu :items="savedFilterItems" :content="{ align: 'start' }">
        <UTooltip :text="t('nav.folders')">
          <UButton
            icon="i-lucide-bookmark"
            :aria-label="t('nav.folders')"
            :color="activeSavedFilter ? 'primary' : 'neutral'"
            :variant="activeSavedFilter ? 'soft' : 'ghost'"
            size="xs"
          />
        </UTooltip>
      </UDropdownMenu>
    </div>

    <div class="flex items-center gap-1 shrink-0">
      <UButton
        icon="i-lucide-sliders-horizontal"
        :aria-label="t('savedFilters.advancedFilter')"
        :color="advancedQuery ? 'primary' : 'neutral'"
        :variant="advancedQuery ? 'soft' : 'ghost'"
        size="xs"
        @click="emit('openAdvancedFilter')"
      />
      <UDropdownMenu :items="sortMenuItems" :content="{ align: 'start' }">
        <UButton
          :icon="currentSort.icon"
          :aria-label="t('conversations.sort.label')"
          color="neutral"
          variant="ghost"
          size="xs"
          :disabled="disabled"
        />
      </UDropdownMenu>
    </div>
  </div>
</template>
