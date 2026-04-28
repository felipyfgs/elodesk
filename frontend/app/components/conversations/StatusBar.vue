<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

defineProps<{
  currentStatus: { label: string, icon: string }
  currentSort: { icon: string }
  statusMenuItems: DropdownMenuItem[]
  sortMenuItems: DropdownMenuItem[]
  filterMenuItems: DropdownMenuItem[]
  advancedQuery: FilterQueryPayload | null
  hasScopeFilter: boolean
}>()

const emit = defineEmits<{
  openAdvancedFilter: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex items-center justify-between gap-1 border-b border-default px-2 py-1">
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
    <div class="flex items-center gap-1">
      <UDropdownMenu :items="filterMenuItems" :content="{ align: 'end' }">
        <UButton
          icon="i-lucide-filter"
          :aria-label="t('conversations.sidebar.title')"
          :color="hasScopeFilter ? 'primary' : 'neutral'"
          :variant="hasScopeFilter ? 'soft' : 'ghost'"
          size="xs"
          :disabled="!!advancedQuery"
        />
      </UDropdownMenu>
      <UButton
        icon="i-lucide-sliders-horizontal"
        :aria-label="t('savedFilters.advancedFilter')"
        :color="advancedQuery ? 'primary' : 'neutral'"
        :variant="advancedQuery ? 'soft' : 'ghost'"
        size="xs"
        @click="emit('openAdvancedFilter')"
      />
      <UDropdownMenu :items="sortMenuItems" :content="{ align: 'end' }">
        <UButton
          :icon="currentSort.icon"
          :aria-label="t('conversations.sort.label')"
          color="neutral"
          variant="ghost"
          size="xs"
          :disabled="!!advancedQuery"
        />
      </UDropdownMenu>
    </div>
  </div>
</template>
