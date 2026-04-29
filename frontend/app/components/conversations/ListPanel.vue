<script setup lang="ts">
import type { DropdownMenuItem, TabsItem } from '@nuxt/ui'
import type { Conversation } from '~/stores/conversations'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'
import { useConversationsStore } from '~/stores/conversations'
import { useInboxesStore } from '~/stores/inboxes'
import { useTeamsStore } from '~/stores/teams'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'

interface FiltersBundle {
  advancedQuery: FilterQueryPayload | null
  activeSavedFilter: SavedFilter | null
  editingFilterId: string | null
  advancedInitialQuery: FilterQueryPayload | null
  advancedInitialName: string
  tabItems: TabsItem[]
  statusMenuItems: DropdownMenuItem[]
  currentStatus: { label: string, icon: string }
  sortMenuItems: DropdownMenuItem[]
  currentSort: { icon: string }
  flagFilterItems: DropdownMenuItem[]
  statusFlagCount: number
}

const props = defineProps<{
  filters: FiltersBundle
  displayedList: Conversation[]
  loading: boolean
}>()

const selected = defineModel<Conversation | null>('selected')
const showAdvancedFilter = defineModel<boolean>('showAdvancedFilter', { default: false })
const activeTab = defineModel<string>('activeTab', { default: '' })

const emit = defineEmits<{
  openAdvancedFilter: []
  clearAdvancedFilter: []
  editActiveFilter: []
  deleteSavedFilter: [id: string]
  advancedApply: [query: FilterQueryPayload]
  advancedSaved: [filter: SavedFilter]
  applySavedFilter: [filter: SavedFilter]
}>()

const { t } = useI18n()
const route = useRoute()
const convs = useConversationsStore()
const inboxes = useInboxesStore()
const teams = useTeamsStore()
const savedFilters = useSavedFiltersStore()

// Prefer reactive access to the bundle (props is reactive in Vue 3.4+).
const f = computed(() => props.filters)

const panelTitle = computed(() => {
  const cf = convs.filters
  const totalScope = (cf.inboxIds?.length ?? 0) + (cf.labelIds?.length ?? 0) + (cf.teamIds?.length ?? 0)

  if (totalScope === 1) {
    if (cf.inboxIds?.length === 1) {
      const inbox = inboxes.list.find(i => i.id === cf.inboxIds![0])
      return inbox?.name ?? t('nav.channels')
    }
    if (cf.labelIds?.length === 1) return `#${cf.labelIds[0]}`
    if (cf.teamIds?.length === 1) {
      const team = teams.list.find(tm => tm.id === cf.teamIds![0])
      return team?.name ?? t('nav.teams')
    }
  }
  if (totalScope > 1) return t('conversations.scopedFilters', { count: totalScope })

  if (route.path.endsWith('/conversations/unattended')) return t('nav.unattended')
  if (route.path.includes('/conversations/filter/')) {
    const id = route.params.id as string | undefined
    const sf = id ? savedFilters.list.find(s => s.id === id) : undefined
    return sf?.name ?? t('nav.folders')
  }
  return t('conversations.title')
})

</script>

<template>
  <UDashboardPanel
    id="conversations-list"
    :default-size="22"
    :min-size="18"
    :max-size="32"
    resizable
  >
    <UDashboardNavbar :title="panelTitle">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
      <template #trailing>
        <UBadge :label="displayedList.length" variant="subtle" />
      </template>
    </UDashboardNavbar>

    <ConversationsStatusBar
      :current-status="f.currentStatus"
      :current-sort="f.currentSort"
      :status-menu-items="f.statusMenuItems"
      :sort-menu-items="f.sortMenuItems"
      :advanced-query="f.advancedQuery"
      :active-saved-filter="f.activeSavedFilter"
      @open-advanced-filter="emit('openAdvancedFilter')"
      @apply-saved-filter="(filter) => emit('applySavedFilter', filter)"
    />

    <ConversationsActiveFilterBanner
      :advanced-query="f.advancedQuery"
      :active-saved-filter="f.activeSavedFilter"
      @clear-advanced-filter="emit('clearAdvancedFilter')"
      @edit-active-filter="emit('editActiveFilter')"
      @delete-saved-filter="(id) => emit('deleteSavedFilter', id)"
    />

    <!--
      Linha de tabs (mine / sem agente / todas) + dropdown ao lado com flags
      ortogonais (Não lidas, Não atendidas) como toggles independentes que
      empilham em cima de qualquer tab.
    -->
    <div class="flex items-center gap-1 border-b border-default px-2 py-1">
      <UTabs
        v-model="activeTab"
        :items="f.tabItems"
        :content="false"
        size="xs"
        class="flex-1 min-w-0"
      />
      <UDropdownMenu :items="f.flagFilterItems" :content="{ align: 'end' }">
        <UTooltip :text="t('conversations.moreFilters')">
          <UButton
            icon="i-lucide-filter"
            :aria-label="t('conversations.moreFilters')"
            :color="f.statusFlagCount ? 'primary' : 'neutral'"
            :variant="f.statusFlagCount ? 'soft' : 'ghost'"
            size="xs"
          >
            <UBadge
              v-if="f.statusFlagCount"
              :label="String(f.statusFlagCount)"
              size="sm"
              color="primary"
              variant="subtle"
            />
          </UButton>
        </UTooltip>
      </UDropdownMenu>
    </div>

    <ConversationsToolbar :total="displayedList.length" />

    <div v-if="loading" class="flex items-center justify-center py-12">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>
    <ConversationsList v-else-if="displayedList.length" v-model="selected" :items="displayedList" />
    <div v-else class="flex flex-col items-center justify-center py-12 gap-2">
      <UIcon name="i-lucide-message-circle-off" class="size-12 text-dimmed" />
      <p class="text-muted text-sm">
        {{ t('conversations.empty') }}
      </p>
    </div>

    <GlobalAudioMiniPlayer />

    <FiltersFilterBuilder
      v-model="showAdvancedFilter"
      filter-type="conversation"
      :initial-query="f.advancedInitialQuery"
      :initial-name="f.advancedInitialName"
      :editing-id="f.editingFilterId"
      @apply="(q) => emit('advancedApply', q)"
      @save="(saved) => emit('advancedSaved', saved)"
    />
  </UDashboardPanel>
</template>
