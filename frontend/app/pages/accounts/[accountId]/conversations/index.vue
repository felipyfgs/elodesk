<script setup lang="ts">
import { breakpointsTailwind, useStorage } from '@vueuse/core'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useTeamsStore, type Team } from '~/stores/teams'
import { useAgentsStore } from '~/stores/agents'
import { useCannedResponsesStore, type CannedResponse } from '~/stores/cannedResponses'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const convs = useConversationsStore()
const inboxes = useInboxesStore()
const labels = useLabelsStore()
const teams = useTeamsStore()
const agents = useAgentsStore()
const canned = useCannedResponsesStore()

const listSize = useStorage('conversations:list-size', 22)

// Selection state
const selected = ref<Conversation | null>(null)
const isPanelOpen = computed({
  get: () => !!selected.value,
  set: (v: boolean) => { if (!v) selected.value = null }
})

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('lg')

// Filters composable
const {
  advancedQuery, showAdvancedFilter, activeSavedFilter, editingFilterId,
  advancedInitialQuery, advancedInitialName, displayedList,
  tabItems, activeTab, statusMenuItems, currentStatus,
  sortMenuItems, currentSort, filterMenuItems,
  hasScopeFilter, activeFilterSummary,
  onAdvancedApply, onAdvancedSaved, editActiveFilter,
  clearAdvancedFilter, openAdvancedFilter,
  fetchSavedFilters, deleteSavedFilter
} = useConversationFilters(load)

// Realtime composable
const { connect: connectRealtime } = useConversationRealtime(selected)

// Data loading
async function loadMeta() {
  if (!auth.account?.id) return
  const params: Record<string, string> = {}
  if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId
  const qs = new URLSearchParams(params).toString()
  const url = `/accounts/${auth.account.id}/conversations/meta${qs ? `?${qs}` : ''}`
  try {
    const res = await api<{ payload: import('~/stores/conversations').ConversationMeta }>(url)
    if (res.payload) convs.setMeta(res.payload)
  } catch (err) {
    if (import.meta.dev) console.warn('[conversations] failed to load meta', err)
  }
}

const STATUS_CODE: Record<string, string> = { OPEN: '0', RESOLVED: '1', PENDING: '2', SNOOZED: '3' }
const ASSIGNEE_TYPE: Record<string, string> = { mine: 'mine', unassigned: 'unassigned', all: 'all', mentions: 'all' }

async function load() {
  if (!auth.account?.id) return

  convs.loading = true
  try {
    if (advancedQuery.value) {
      const res = await api<{ payload: Conversation[] }>(
        `/accounts/${auth.account.id}/conversations/filter`,
        { method: 'POST', body: { query: advancedQuery.value, page: 1, per_page: 100 } }
      )
      if (res.payload) convs.setAll(res.payload)
    } else {
      const params: Record<string, string> = {
        page: '1',
        per_page: '100',
        sort_by: convs.filters.sortBy
      }
      if (convs.filters.status) params.status = STATUS_CODE[convs.filters.status]
      if (convs.filters.inboxId) params.inbox_id = convs.filters.inboxId
      const assigneeType = ASSIGNEE_TYPE[convs.filters.tab]
      if (assigneeType && assigneeType !== 'all') params.assignee_type = assigneeType

      const res = await api<{ payload: Conversation[] }>(`/accounts/${auth.account.id}/conversations?${new URLSearchParams(params).toString()}`)
      if (res.payload) convs.setAll(res.payload)
    }
  } finally {
    convs.loading = false
  }

  loadMeta()

  // Load supporting data in parallel
  const promises: Promise<void>[] = []
  if (!inboxes.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/inboxes`)
        .then((res) => { if (res.payload) inboxes.setAll(res.payload as Inbox[]) })
        .catch(() => {})
    )
  }
  if (!labels.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/labels`)
        .then((res) => { if (res.payload) labels.setAll(res.payload as Label[]) })
        .catch(() => {})
    )
  }
  if (!teams.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/teams`)
        .then((res) => { if (res.payload) teams.setAll(res.payload as Team[]) })
        .catch(() => {})
    )
  }
  if (!agents.items.length) {
    promises.push(agents.fetch().catch(() => {}))
  }
  promises.push(fetchSavedFilters())
  if (!canned.list.length) {
    promises.push(
      api<{ payload: unknown[] }>(`/accounts/${auth.account.id}/canned_responses`)
        .then((res) => { if (res.payload) canned.setAll(res.payload as CannedResponse[]) })
        .catch(() => {})
    )
  }
  await Promise.allSettled(promises)
}

onMounted(async () => {
  await load()
  connectRealtime()
})

watch(displayedList, () => {
  if (!displayedList.value.find(c => c.id === selected.value?.id)) {
    selected.value = null
  }
})

watch(() => convs.filters, () => {
  if (!advancedQuery.value) load()
}, { deep: true })
</script>

<template>
  <UDashboardPanel
    id="conversations-list"
    :default-size="listSize"
    :min-size="15"
    :max-size="35"
    resizable
  >
    <UDashboardNavbar :title="t('conversations.title')">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
      <template #trailing>
        <UBadge :label="displayedList.length" variant="subtle" />
      </template>
      <template #right>
        <UDropdownMenu :items="statusMenuItems">
          <UButton
            :label="currentStatus.label"
            :icon="currentStatus.icon"
            trailing-icon="i-lucide-chevrons-up-down"
            color="neutral"
            variant="ghost"
            size="sm"
            :disabled="!!advancedQuery"
          />
        </UDropdownMenu>
        <UDropdownMenu :items="filterMenuItems" :content="{ align: 'start' }">
          <UButton
            icon="i-lucide-filter"
            :aria-label="t('conversations.sidebar.title')"
            :color="hasScopeFilter ? 'primary' : 'neutral'"
            :variant="hasScopeFilter ? 'soft' : 'ghost'"
            size="sm"
            :disabled="!!advancedQuery"
          />
        </UDropdownMenu>
        <UButton
          icon="i-lucide-sliders-horizontal"
          :aria-label="t('savedFilters.advancedFilter')"
          :color="advancedQuery ? 'primary' : 'neutral'"
          :variant="advancedQuery ? 'soft' : 'ghost'"
          size="sm"
          @click="openAdvancedFilter"
        />
        <UDropdownMenu :items="sortMenuItems" :content="{ align: 'start' }">
          <UButton
            :icon="currentSort.icon"
            :aria-label="t('conversations.sort.label')"
            color="neutral"
            variant="ghost"
            size="sm"
            :disabled="!!advancedQuery"
          />
        </UDropdownMenu>
      </template>
    </UDashboardNavbar>

    <div
      v-if="hasScopeFilter && !advancedQuery"
      class="flex items-center justify-between gap-2 px-3 py-1.5 bg-primary/5 border-b border-default text-xs"
    >
      <div class="flex items-center gap-1.5 text-muted min-w-0">
        <UIcon name="i-lucide-filter" class="size-3.5 shrink-0" />
        <span class="truncate">{{ activeFilterSummary }}</span>
      </div>
      <UButton
        :label="t('conversations.sidebar.clearFilters')"
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        @click="navigateTo(`/accounts/${auth.account?.id}/conversations`)"
      />
    </div>

    <div
      v-if="advancedQuery"
      class="flex items-center justify-between gap-2 px-3 py-1.5 bg-primary/5 border-b border-default text-xs"
    >
      <div class="flex items-center gap-1.5 text-muted min-w-0">
        <UIcon name="i-lucide-sliders-horizontal" class="size-3.5 shrink-0" />
        <span class="truncate">
          {{ activeSavedFilter?.name ?? t('savedFilters.advancedFilterActive', { count: advancedQuery.conditions.length }) }}
        </span>
      </div>
      <div class="flex items-center gap-1">
        <UButton
          :label="t('savedFilters.edit')"
          icon="i-lucide-pencil"
          color="neutral"
          variant="ghost"
          size="xs"
          @click="editActiveFilter"
        />
        <UButton
          v-if="activeSavedFilter"
          :label="t('savedFilters.delete')"
          icon="i-lucide-trash-2"
          color="error"
          variant="ghost"
          size="xs"
          @click="activeSavedFilter && deleteSavedFilter(activeSavedFilter.id)"
        />
        <UButton
          :label="t('savedFilters.clearFilter')"
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="xs"
          @click="clearAdvancedFilter"
        />
      </div>
    </div>

    <div class="border-b border-default px-2 py-1">
      <UTabs
        v-model="activeTab"
        :items="tabItems"
        :content="false"
        size="xs"
        class="w-full"
      />
    </div>

    <ConversationsToolbar />

    <div v-if="convs.hasSelection" class="flex items-center justify-between px-4 py-2 bg-primary/5 border-b border-default">
      <label class="flex items-center gap-2 text-sm cursor-pointer">
        <UCheckbox
          :model-value="convs.selection.length === displayedList.length && displayedList.length > 0"
          @update:model-value="(v: boolean | string) => v ? convs.selectAll() : convs.clearSelection()"
        />
        <span class="text-muted">{{ t('conversations.bulk.selectAll') }}</span>
      </label>
    </div>

    <div v-if="convs.loading" class="flex items-center justify-center py-12">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>
    <ConversationsList v-else-if="displayedList.length" v-model="selected" :items="displayedList" />
    <div v-else class="flex flex-col items-center justify-center py-12 gap-2">
      <UIcon name="i-lucide-message-circle-off" class="size-12 text-dimmed" />
      <p class="text-muted text-sm">
        {{ t('conversations.empty') }}
      </p>
    </div>

    <FiltersFilterBuilder
      v-model="showAdvancedFilter"
      filter-type="conversation"
      :initial-query="advancedInitialQuery"
      :initial-name="advancedInitialName"
      :editing-id="editingFilterId"
      @apply="onAdvancedApply"
      @save="onAdvancedSaved"
    />
  </UDashboardPanel>

  <ConversationsConversationThread v-if="selected && !isMobile" :conversation="selected" @close="selected = null" />
  <div v-else-if="!isMobile" class="hidden lg:flex flex-1 items-center justify-center flex-col gap-2">
    <UIcon name="i-lucide-inbox" class="size-32 text-dimmed" />
    <p class="text-muted">
      {{ t('conversations.select') }}
    </p>
  </div>

  <ClientOnly>
    <USlideover v-if="isMobile" v-model:open="isPanelOpen">
      <template #content>
        <ConversationsConversationThread v-if="selected" :conversation="selected" @close="selected = null" />
      </template>
    </USlideover>
  </ClientOnly>
</template>
