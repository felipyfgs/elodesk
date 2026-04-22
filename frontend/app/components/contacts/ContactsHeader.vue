<script setup lang="ts">
const props = defineProps<{
  title: string
  searchValue?: string
  hasActiveFilters?: boolean
}>()

const emit = defineEmits<{
  search: [value: string]
  filter: []
  import: []
  add: []
}>()

const { t } = useI18n()

const searchInput = ref(props.searchValue || '')

watch(() => props.searchValue, (val) => {
  searchInput.value = val || ''
})

function handleSearch(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('search', value)
}
</script>

<template>
  <div class="sticky top-0 z-10 bg-default border-b border-default">
    <div class="px-6 py-4 max-w-5xl mx-auto">
      <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <!-- Título -->
        <h1 class="text-xl font-semibold">
          {{ title }}
        </h1>

        <!-- Ações -->
        <div class="flex flex-wrap items-center gap-2 w-full sm:w-auto">
          <!-- Busca -->
          <div class="relative flex-1 sm:flex-initial sm:w-64">
            <UInput
              :model-value="searchInput"
              icon="i-lucide-search"
              :placeholder="t('contacts.search')"
              @input="handleSearch"
            />
          </div>

          <!-- Filtros -->
          <div class="relative">
            <UButton
              icon="i-lucide-list-filter"
              color="neutral"
              variant="outline"
              @click="emit('filter')"
            />
            <div
              v-if="hasActiveFilters"
              class="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-primary"
            />
          </div>

          <!-- Importar -->
          <UButton
            :label="t('contacts.importCsv')"
            icon="i-lucide-upload"
            color="neutral"
            variant="outline"
            @click="emit('import')"
          />

          <!-- Adicionar -->
          <UButton
            :label="t('contacts.add')"
            icon="i-lucide-plus"
            @click="emit('add')"
          />
        </div>
      </div>
    </div>
  </div>
</template>
