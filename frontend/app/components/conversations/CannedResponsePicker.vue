<script setup lang="ts">
import { useCannedResponsesStore } from '~/stores/cannedResponses'

const props = defineProps<{
  search?: string
}>()

const emit = defineEmits<{
  select: [content: string]
}>()

const { t } = useI18n()
const canned = useCannedResponsesStore()

const filtered = computed(() => {
  const q = (props.search ?? '').trim()
  if (!q) return canned.list.slice(0, 8)
  return canned.search(q).slice(0, 8)
})
</script>

<template>
  <div class="w-72 py-1">
    <p class="px-3 py-1 text-[10px] font-medium uppercase tracking-wider text-dimmed">
      {{ t('conversations.compose.canned') }}
    </p>
    <ul class="max-h-60 overflow-y-auto">
      <li v-for="item in filtered" :key="item.id">
        <button
          type="button"
          class="w-full px-3 py-2 text-left transition-colors hover:bg-elevated"
          @click="emit('select', item.content)"
        >
          <span class="block font-mono text-xs text-primary">/{{ item.shortCode }}</span>
          <span class="mt-0.5 block truncate text-xs text-muted">{{ item.content }}</span>
        </button>
      </li>
    </ul>
    <p v-if="!filtered.length" class="px-3 py-3 text-center text-xs text-muted">
      {{ t('conversations.compose.cannedEmpty') }}
    </p>
  </div>
</template>
