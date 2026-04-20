<script setup lang="ts">
import { useConversationsStore } from '~/stores/conversations'
import { useInboxesStore } from '~/stores/inboxes'
import { useLabelsStore } from '~/stores/labels'
import { useTeamsStore } from '~/stores/teams'

const { t } = useI18n()
const convs = useConversationsStore()
const inboxes = useInboxesStore()
const labels = useLabelsStore()
const teams = useTeamsStore()

const route = useRoute()

function setTab(tab: string) {
  convs.setFilters({ tab: tab as 'mine' | 'unassigned' | 'all' | 'mentions' })
  navigateTo({ query: { ...route.query, tab } })
}

function setInboxFilter(inboxId: string) {
  if (convs.filters.inboxId === inboxId) {
    convs.setFilters({ inboxId: undefined })
  } else {
    convs.setFilters({ inboxId })
  }
}

function setLabelFilter(labelId: string) {
  if (convs.filters.labelId === labelId) {
    convs.setFilters({ labelId: undefined })
  } else {
    convs.setFilters({ labelId })
  }
}

function setTeamFilter(teamId: string) {
  if (convs.filters.teamId === teamId) {
    convs.setFilters({ teamId: undefined })
  } else {
    convs.setFilters({ teamId })
  }
}

function setStatusFilter(status: string) {
  if (convs.filters.status === status) {
    convs.setFilters({ status: undefined })
  } else {
    convs.setFilters({ status })
  }
}
</script>

<template>
  <div class="p-4 flex flex-col gap-4 overflow-y-auto h-full">
    <!-- Tab filters -->
    <div class="flex flex-col gap-1">
      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors"
        :class="convs.filters.tab === 'mine' ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setTab('mine')"
      >
        <UIcon name="i-lucide-user" class="size-4" />
        {{ t('conversations.sidebar.mine') }}
      </button>
      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors"
        :class="convs.filters.tab === 'unassigned' ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setTab('unassigned')"
      >
        <UIcon name="i-lucide-user-x" class="size-4" />
        {{ t('conversations.sidebar.unassigned') }}
      </button>
      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors"
        :class="convs.filters.tab === 'all' ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setTab('all')"
      >
        <UIcon name="i-lucide-inbox" class="size-4" />
        {{ t('conversations.sidebar.all') }}
      </button>
      <button
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors"
        :class="convs.filters.tab === 'mentions' ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setTab('mentions')"
      >
        <UIcon name="i-lucide-at-sign" class="size-4" />
        {{ t('conversations.sidebar.mentions') }}
      </button>
    </div>

    <UDivider />

    <!-- Status filters -->
    <div class="flex flex-col gap-1">
      <p class="text-xs font-medium text-dimmed uppercase tracking-wider px-3 mb-1">
        {{ t('conversations.sidebar.status') }}
      </p>
      <button
        v-for="s in ['OPEN', 'PENDING', 'RESOLVED', 'SNOOZED']"
        :key="s"
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors"
        :class="convs.filters.status === s ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setStatusFilter(s)"
      >
        {{ t(`conversations.filters.${s.toLowerCase()}`) }}
      </button>
    </div>

    <UDivider />

    <!-- Inboxes -->
    <div class="flex flex-col gap-1">
      <p class="text-xs font-medium text-dimmed uppercase tracking-wider px-3 mb-1">
        {{ t('conversations.sidebar.inboxes') }}
      </p>
      <button
        v-for="inbox in inboxes.list"
        :key="inbox.id"
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors truncate"
        :class="convs.filters.inboxId === inbox.id ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setInboxFilter(inbox.id)"
      >
        {{ inbox.name }}
      </button>
    </div>

    <UDivider />

    <!-- Labels -->
    <div class="flex flex-col gap-1">
      <p class="text-xs font-medium text-dimmed uppercase tracking-wider px-3 mb-1">
        {{ t('conversations.sidebar.labels') }}
      </p>
      <button
        v-for="label in labels.list"
        :key="label.id"
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors truncate"
        :class="convs.filters.labelId === label.id ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setLabelFilter(label.id)"
      >
        <span class="w-2.5 h-2.5 rounded-full shrink-0" :style="{ backgroundColor: label.color }" />
        {{ label.title }}
      </button>
    </div>

    <UDivider />

    <!-- Teams -->
    <div class="flex flex-col gap-1">
      <p class="text-xs font-medium text-dimmed uppercase tracking-wider px-3 mb-1">
        {{ t('conversations.sidebar.teams') }}
      </p>
      <button
        v-for="team in teams.list"
        :key="team.id"
        type="button"
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors truncate"
        :class="convs.filters.teamId === team.id ? 'bg-primary/10 text-primary font-medium' : 'text-toned hover:bg-elevated'"
        @click="setTeamFilter(team.id)"
      >
        {{ team.name }}
      </button>
    </div>
  </div>
</template>
