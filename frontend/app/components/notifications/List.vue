<script setup lang="ts">
import { useNotificationsStore, type Notification } from '~/stores/notifications'

const props = defineProps<{
  emptyTitle?: string
  emptyDescription?: string
  selectedId?: number | null
}>()
const emit = defineEmits<{
  select: [n: Notification]
  toggleRead: [n: Notification]
}>()

const { t } = useI18n()
const store = useNotificationsStore()

const hasMore = computed(() => store.cursor > 0)
const loadingMore = ref(false)

function onItemClick(n: Notification) {
  emit('select', n)
}

async function onLoadMore() {
  loadingMore.value = true
  try {
    await store.fetchMore()
  } finally {
    loadingMore.value = false
  }
}

const emptyTitle = computed(() => props.emptyTitle ?? t('notifications.empty'))
const emptyDescription = computed(() => props.emptyDescription ?? '')

// Auto-scroll the active row into view when the parent navigates by keyboard.
const itemRefs = ref<Record<number, Element | null>>({})
watch(() => props.selectedId, (id) => {
  if (!id) return
  nextTick(() => {
    const el = itemRefs.value[id]
    if (el) el.scrollIntoView({ block: 'nearest' })
  })
})
</script>

<template>
  <div v-if="store.isLoading && store.items.length === 0" class="flex flex-col gap-1 p-3">
    <USkeleton v-for="n in 6" :key="n" class="h-16 w-full rounded-md" />
  </div>

  <UEmpty
    v-else-if="store.items.length === 0"
    icon="i-lucide-bell-off"
    :title="emptyTitle"
    :description="emptyDescription"
    variant="naked"
    class="py-12"
  />

  <ul
    v-else
    class="min-h-0 flex-1 divide-y divide-default overflow-y-auto"
    role="listbox"
    :aria-label="t('notifications.title')"
  >
    <li
      v-for="n in store.items"
      :key="n.id"
      :ref="(el) => { itemRefs[n.id] = el as Element | null }"
    >
      <NotificationsItem
        :notification="n"
        :selected="selectedId === n.id"
        @click="onItemClick"
        @toggle-read="(notification) => emit('toggleRead', notification)"
      />
    </li>

    <li v-if="hasMore" class="flex justify-center p-3">
      <UButton
        variant="outline"
        color="neutral"
        icon="i-lucide-chevron-down"
        :loading="loadingMore"
        size="sm"
        @click="onLoadMore"
      >
        {{ t('notifications.loadMore') }}
      </UButton>
    </li>
  </ul>
</template>
