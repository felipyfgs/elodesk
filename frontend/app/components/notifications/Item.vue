<script setup lang="ts">
import { formatTimeAgo } from '@vueuse/core'
import type { Notification } from '~/stores/notifications'

const props = defineProps<{ notification: Notification }>()
const emit = defineEmits<{ click: [n: Notification] }>()

const link = computed(() => {
  const p = props.notification.payload as Record<string, unknown> | undefined
  const deep = p?.deep_link
  return typeof deep === 'string' ? deep : undefined
})

const title = computed(() => {
  const p = props.notification.payload as Record<string, unknown> | undefined
  return (p?.title as string | undefined) ?? props.notification.type
})

const body = computed(() => {
  const p = props.notification.payload as Record<string, unknown> | undefined
  return (p?.body as string | undefined) ?? (p?.message as string | undefined) ?? ''
})

function onClick() {
  emit('click', props.notification)
}
</script>

<template>
  <component
    :is="link ? 'NuxtLink' : 'div'"
    :to="link"
    class="flex items-start gap-3 py-3 px-2 rounded-md hover:bg-muted cursor-pointer"
    @click="onClick"
  >
    <UChip
      :show="!props.notification.readAt"
      color="primary"
      size="sm"
      inset
    >
      <UAvatar size="sm" :alt="title" />
    </UChip>
    <div class="flex-1 min-w-0">
      <p class="text-sm font-medium truncate">
        {{ title }}
      </p>
      <p v-if="body" class="text-xs text-muted line-clamp-2">
        {{ body }}
      </p>
      <p class="text-[11px] text-muted mt-0.5">
        {{ formatTimeAgo(new Date(props.notification.createdAt)) }}
      </p>
    </div>
  </component>
</template>
