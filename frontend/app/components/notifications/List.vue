<script setup lang="ts">
import { useNotificationsStore, type Notification } from '~/stores/notifications'

const props = defineProps<{
  emptyTitle?: string
  emptyDescription?: string
}>()

const { t } = useI18n()
const store = useNotificationsStore()

const hasMore = computed(() => store.cursor > 0)
const loadingMore = ref(false)

async function onItemClick(n: Notification) {
  if (!n.readAt) await store.markRead(n.id)
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
</script>

<template>
  <div class="flex flex-col gap-3">
    <div v-if="store.loading && store.items.length === 0" class="flex flex-col gap-1">
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

    <template v-else>
      <UPageList divide>
        <NotificationsItem
          v-for="n in store.items"
          :key="n.id"
          :notification="n"
          @click="onItemClick"
        />
      </UPageList>

      <div v-if="hasMore" class="flex justify-center">
        <UButton
          variant="outline"
          color="neutral"
          icon="i-lucide-chevron-down"
          :loading="loadingMore"
          @click="onLoadMore"
        >
          {{ t('notifications.loadMore') }}
        </UButton>
      </div>
    </template>
  </div>
</template>
