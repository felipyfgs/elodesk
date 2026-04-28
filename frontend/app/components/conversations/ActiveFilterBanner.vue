<script setup lang="ts">
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'
import type { SavedFilter } from '~/stores/savedFilters'

defineProps<{
  hasScopeFilter: boolean
  activeFilterSummary: string
  advancedQuery: FilterQueryPayload | null
  activeSavedFilter: SavedFilter | null
}>()

const emit = defineEmits<{
  clearScopeFilter: []
  clearAdvancedFilter: []
  editActiveFilter: []
  deleteSavedFilter: [id: string]
}>()

const { t } = useI18n()
</script>

<template>
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
      @click="emit('clearScopeFilter')"
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
        @click="emit('editActiveFilter')"
      />
      <UButton
        v-if="activeSavedFilter"
        :label="t('savedFilters.delete')"
        icon="i-lucide-trash-2"
        color="error"
        variant="ghost"
        size="xs"
        @click="activeSavedFilter && emit('deleteSavedFilter', activeSavedFilter.id)"
      />
      <UButton
        :label="t('savedFilters.clearFilter')"
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        @click="emit('clearAdvancedFilter')"
      />
    </div>
  </div>
</template>
