<script setup lang="ts">
import type { DropdownMenuItem, TabsItem } from '@nuxt/ui'
import type { Conversation } from '~/stores/conversations'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'
import type { SavedFilter } from '~/stores/savedFilters'

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
  filterMenuItems: DropdownMenuItem[]
  hasScopeFilter: boolean
  activeFilterSummary: string
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
  clearScopeFilter: []
  clearAdvancedFilter: []
  editActiveFilter: []
  deleteSavedFilter: [id: string]
  advancedApply: [query: FilterQueryPayload]
  advancedSaved: [filter: SavedFilter]
}>()

const { t } = useI18n()

// Prefer reactive access to the bundle (props is reactive in Vue 3.4+).
const f = computed(() => props.filters)
</script>

<template>
  <UDashboardPanel
    id="conversations-list"
    :default-size="22"
    :min-size="22"
    :max-size="22"
  >
    <UDashboardNavbar :title="t('conversations.title')">
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
      :filter-menu-items="f.filterMenuItems"
      :advanced-query="f.advancedQuery"
      :has-scope-filter="f.hasScopeFilter"
      @open-advanced-filter="emit('openAdvancedFilter')"
    />

    <ConversationsActiveFilterBanner
      :has-scope-filter="f.hasScopeFilter"
      :active-filter-summary="f.activeFilterSummary"
      :advanced-query="f.advancedQuery"
      :active-saved-filter="f.activeSavedFilter"
      @clear-scope-filter="emit('clearScopeFilter')"
      @clear-advanced-filter="emit('clearAdvancedFilter')"
      @edit-active-filter="emit('editActiveFilter')"
      @delete-saved-filter="(id) => emit('deleteSavedFilter', id)"
    />

    <div class="border-b border-default px-2 py-1">
      <UTabs
        v-model="activeTab"
        :items="f.tabItems"
        :content="false"
        size="xs"
        class="w-full"
      />
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
