<script setup lang="ts">
const props = defineProps<{
  currentPage: number
  totalItems: number
  itemsPerPage?: number
  hasMore?: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:page': [page: number]
  'load-more': []
}>()

const { t } = useI18n()

const itemsPerPage = computed(() => props.itemsPerPage || 25)

const startItem = computed(() => {
  return (props.currentPage - 1) * itemsPerPage.value + 1
})

const endItem = computed(() => {
  return Math.min(props.currentPage * itemsPerPage.value, props.totalItems)
})
</script>

<template>
  <div class="px-6 py-4 max-w-5xl mx-auto">
    <div class="flex items-center justify-between gap-4">
      <!-- Contador -->
      <div class="text-sm text-muted">
        {{ t('contacts.pagination.showing', { start: startItem, end: endItem, total: totalItems }) }}
      </div>

      <!-- Paginação -->
      <UPagination
        :model-value="currentPage"
        :total="totalItems"
        :items-per-page="itemsPerPage"
        @update:model-value="emit('update:page', $event)"
      />
    </div>
  </div>
</template>
