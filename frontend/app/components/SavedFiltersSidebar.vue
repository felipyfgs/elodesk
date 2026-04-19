<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { useSavedFiltersStore } from '~/stores/savedFilters'

const props = defineProps<{
  filterType: 'conversation' | 'contact'
}>()

const emit = defineEmits<{
  apply: [filter: any]
}>()

const { t } = useI18n()
const api = useApi()
const store = useSavedFiltersStore()

const showBuilder = ref(false)
const loading = ref(false)
const activeId = ref<string | null>(null)

const filters = computed(() =>
  props.filterType === 'conversation' ? store.conversationFilters : store.contactFilters
)

async function applyFilter(filter: any) {
  activeId.value = filter.id
  emit('apply', filter)
}

async function removeFilter(id: string) {
  loading.value = true
  try {
    await api(`/saved-filters/${id}`, { method: 'DELETE' })
    store.remove(id)
    if (activeId.value === id) activeId.value = null
  } finally {
    loading.value = false
  }
}

function onSaved() {
  void fetchFilters()
}

async function fetchFilters() {
  loading.value = true
  try {
    const list = await api<any[]>('/saved-filters', {
      params: { filter_type: props.filterType }
    })
    store.setAll([...store.list.filter(f => f.filterType !== props.filterType), ...list])
  } finally {
    loading.value = false
  }
}

onMounted(fetchFilters)

const menuItems = computed<(id: string) => DropdownMenuItem[][]>(() => (id: string) => [
  [{
    label: t('savedFilters.delete'),
    icon: 'i-lucide-trash',
    color: 'error' as const,
    onSelect: () => removeFilter(id)
  }]
])
</script>

<template>
  <div class="space-y-1">
    <div class="flex items-center justify-between px-2 py-1">
      <span class="text-xs font-medium uppercase tracking-wider text-dimmed">
        {{ t('savedFilters.title') }}
      </span>
      <UButton
        icon="i-lucide-plus"
        size="xs"
        color="neutral"
        variant="ghost"
        :label="t('savedFilters.create')"
        @click="showBuilder = true"
      />
    </div>

    <div v-if="!filters.length" class="px-2 py-1">
      <p class="text-xs text-muted">{{ t('savedFilters.empty') }}</p>
    </div>

    <div v-for="filter in filters" :key="filter.id" class="group relative">
      <button
        class="w-full text-left px-2 py-1.5 rounded-md text-sm transition-colors"
        :class="activeId === filter.id ? 'bg-elevated text-highlighted font-medium' : 'hover:bg-elevated text-muted hover:text-highlighted'"
        @click="applyFilter(filter)"
      >
        {{ filter.name }}
      </button>
      <UDropdownMenu
        :items="menuItems(filter.id)"
        :content="{ align: 'end', collisionPadding: 8 }"
      >
        <UButton
          icon="i-lucide-ellipsis"
          size="xs"
          color="neutral"
          variant="ghost"
          class="absolute right-1 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity"
        />
      </UDropdownMenu>
    </div>

    <FilterBuilder
      v-model="showBuilder"
      :filter-type="filterType"
      @save="onSaved"
    />
  </div>
</template>
