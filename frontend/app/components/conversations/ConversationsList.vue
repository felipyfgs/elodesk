<script setup lang="ts">
import { format, isToday } from 'date-fns'
import type { Conversation } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'

const props = defineProps<{
  items: Conversation[]
}>()

const selected = defineModel<Conversation | null>()
const convs = useConversationsStore()

const itemRefs = ref<Record<string, Element | null>>({})

function contactName(c: Conversation): string {
  return c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber || c.contactInbox?.contact?.waJid || '—'
}

function lastMessage(c: Conversation): string {
  const msg = c.meta?.lastNonActivityMessage
  if (!msg) return ''
  if (msg.attachments?.length) return `[${msg.attachments.length} attachment(s)]`
  return msg.content || ''
}

watch(selected, () => {
  if (!selected.value) return
  const ref = itemRefs.value[selected.value.id]
  if (ref) ref.scrollIntoView({ block: 'nearest' })
})

defineShortcuts({
  arrowdown: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    if (idx === -1) selected.value = props.items[0] ?? null
    else if (idx < props.items.length - 1) selected.value = props.items[idx + 1] ?? null
  },
  arrowup: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    if (idx === -1) selected.value = props.items.at(-1) ?? null
    else if (idx > 0) selected.value = props.items[idx - 1] ?? null
  }
})
</script>

<template>
  <div class="overflow-y-auto divide-y divide-default">
    <div
      v-for="c in items"
      :key="c.id"
      :ref="(el) => { itemRefs[c.id] = el as Element | null }"
      class="p-4 sm:px-6 text-sm cursor-pointer border-l-2 transition-colors"
      :class="[
        c.status === 0 ? 'text-highlighted' : 'text-toned',
        selected && selected.id === c.id
          ? 'border-primary bg-primary/10'
          : 'border-bg hover:border-primary hover:bg-primary/5'
      ]"
      @click="selected = c"
    >
      <div class="flex items-center gap-2">
        <!-- Selection checkbox -->
        <UCheckbox
          :model-value="convs.selection.includes(c.id)"
          @update:model-value="() => convs.toggleSelection(c.id)"
          @click.stop
        />

        <!-- Contact avatar -->
        <UAvatar
          :alt="contactName(c)"
          :src="c.meta?.sender?.thumbnail ?? undefined"
          size="sm"
        />

        <div class="flex-1 min-w-0">
          <div class="flex items-center justify-between">
            <span class="truncate font-medium">
              {{ contactName(c) }}
            </span>
            <span class="text-xs text-muted shrink-0">
              {{ isToday(new Date(c.lastActivityAt)) ? format(new Date(c.lastActivityAt), 'HH:mm') : format(new Date(c.lastActivityAt), 'dd MMM') }}
            </span>
          </div>

          <div class="flex items-center gap-2 mt-0.5">
            <!-- Inbox badge -->
            <span v-if="c.inbox" class="text-[10px] text-dimmed bg-elevated rounded px-1.5 py-0.5 truncate max-w-[100px]">
              {{ c.inbox.name }}
            </span>

            <!-- Labels -->
            <span
              v-for="label in (c.labels || []).slice(0, 2)"
              :key="label.id"
              class="text-[10px] rounded px-1.5 py-0.5 truncate max-w-[80px]"
              :style="{ backgroundColor: label.color + '20', color: label.color }"
            >
              {{ label.title }}
            </span>
          </div>

          <!-- Last message preview -->
          <p class="text-dimmed line-clamp-1 mt-1 text-xs">
            {{ lastMessage(c) }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
