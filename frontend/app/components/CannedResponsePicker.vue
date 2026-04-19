<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import { useCannedResponsesStore } from '~/stores/cannedResponses'

const props = defineProps<{
  modelValue: boolean
  composerRef?: any
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  insert: [content: string]
}>()

const { t } = useI18n()
const store = useCannedResponsesStore()

const searchTerm = ref('')
const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

watch(() => props.modelValue, (v) => {
  if (!v) searchTerm.value = ''
})

const filtered = computed(() => store.search(searchTerm.value))

const items = computed<DropdownMenuItem[][]>(() => {
  if (!filtered.value.length) return [[{ label: t('common.noResults'), disabled: true }]]
  return filtered.value.map(r => [{
    label: r.shortCode,
    onSelect: () => select(r)
  }])
})

function select(response: { content: string }) {
  emit('insert', response.content)
  isOpen.value = false
}
</script>

<template>
  <UModal v-model:open="isOpen" :title="t('cannedResponses.title')">
    <template #body>
      <UInput
        v-model="searchTerm"
        :placeholder="t('common.search')"
        icon="i-lucide-search"
        autofocus
      />
      <div class="mt-3 max-h-64 overflow-y-auto space-y-1">
        <button
          v-for="item in filtered"
          :key="item.id"
          class="w-full text-left px-3 py-2 rounded-md hover:bg-elevated text-sm"
          @click="select(item)"
        >
          <UBadge variant="subtle" size="xs" class="mb-1">{{ item.shortCode }}</UBadge>
          <p class="text-dimmed truncate">{{ item.content }}</p>
        </button>
        <p v-if="!filtered.length" class="text-sm text-muted p-2">{{ t('common.noResults') }}</p>
      </div>
    </template>
  </UModal>
</template>
