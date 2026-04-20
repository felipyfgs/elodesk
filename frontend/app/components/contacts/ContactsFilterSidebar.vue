<script setup lang="ts">
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'apply': [filter: SavedFilter]
}>()

const { t } = useI18n()
const store = useSavedFiltersStore()

const isOpen = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v)
})

const contactFilters = computed(() => store.contactFilters)

function applyFilter(filter: SavedFilter) {
  emit('apply', filter)
  isOpen.value = false
}
</script>

<template>
  <USlideover v-model:open="isOpen" :title="t('contacts.tabs.overview')">
    <template #body>
      <div class="space-y-4 p-4">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium uppercase tracking-wider text-dimmed">
            {{ t('savedFilters.title') }}
          </span>
        </div>

        <div v-if="!contactFilters.length" class="text-sm text-muted">
          {{ t('savedFilters.empty') }}
        </div>

        <div v-for="filter in contactFilters" :key="filter.id" class="space-y-1">
          <button
            class="w-full text-left px-3 py-2 rounded-md text-sm hover:bg-elevated transition-colors"
            @click="applyFilter(filter)"
          >
            {{ filter.name }}
          </button>
        </div>
      </div>
    </template>
  </USlideover>
</template>
