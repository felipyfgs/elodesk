<script setup lang="ts">
import { format, isToday } from 'date-fns'
import type { Conversation } from '~/stores/conversations'

const props = defineProps<{
  items: Conversation[]
}>()

const selected = defineModel<Conversation | null>()

function title(c: Conversation) {
  return c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}

defineShortcuts({
  arrowdown: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    selected.value = idx === -1 ? props.items[0] ?? null : (idx < props.items.length - 1 ? props.items[idx + 1] ?? null : selected.value)
  },
  arrowup: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    selected.value = idx === -1 ? props.items.at(-1) ?? null : (idx > 0 ? props.items[idx - 1] ?? null : selected.value)
  }
})
</script>

<template>
  <div class="overflow-y-auto divide-y divide-default">
    <button
      v-for="c in items"
      :key="c.id"
      type="button"
      class="w-full text-left p-4 sm:px-6 text-sm cursor-pointer border-l-2 transition-colors"
      :class="[
        c.unreadCount > 0 ? 'text-highlighted' : 'text-toned',
        selected && selected.id === c.id
          ? 'border-primary bg-primary/10'
          : 'border-bg hover:border-primary hover:bg-primary/5'
      ]"
      @click="selected = c"
    >
      <div class="flex items-center justify-between" :class="[c.unreadCount > 0 && 'font-semibold']">
        <div class="flex items-center gap-3 truncate">
          <span class="truncate">{{ title(c) }}</span>
          <UChip v-if="c.unreadCount > 0" />
        </div>
        <span class="text-xs text-muted shrink-0">
          {{ isToday(new Date(c.lastActivityAt)) ? format(new Date(c.lastActivityAt), 'HH:mm') : format(new Date(c.lastActivityAt), 'dd MMM') }}
        </span>
      </div>
      <p class="text-dimmed line-clamp-1 mt-1 text-xs">
        {{ c.inbox?.name }}
      </p>
    </button>
  </div>
</template>
