<script setup lang="ts">
import { useCannedResponsesStore } from '~/stores/cannedResponses'

const props = defineProps<{
  visible?: boolean
  search: string
}>()

const model = defineModel<boolean>()

const emit = defineEmits<{
  select: [content: string]
}>()

const canned = useCannedResponsesStore()

const filtered = computed(() => {
  if (!props.search) return canned.list.slice(0, 8)
  return canned.search(props.search).slice(0, 8)
})

function handleSelect(item: { content: string }) {
  emit('select', item.content)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') model.value = false
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed z-50 bg-popover border border-default rounded-lg shadow-lg py-1 max-h-60 overflow-y-auto w-64"
      tabindex="-1"
      @keydown="handleKeydown"
    >
      <p class="px-3 py-1 text-xs text-dimmed font-medium uppercase tracking-wider">
        Canned Responses
      </p>
      <button
        v-for="item in filtered"
        :key="item.id"
        type="button"
        class="w-full text-left px-3 py-2 text-sm hover:bg-elevated transition-colors"
        @click="handleSelect(item)"
      >
        <span class="font-mono text-primary text-xs">{{ item.shortCode }}</span>
        <p class="text-dimmed text-xs truncate mt-0.5">
          {{ item.content }}
        </p>
      </button>
      <p v-if="!filtered.length" class="px-3 py-2 text-sm text-muted">
        No results
      </p>
    </div>
  </Teleport>
</template>
