<script setup lang="ts">
import type { TimelineItem } from '@nuxt/ui'

interface TimelineEvent {
  id: string
  type: string
  description: string
  createdAt: string
}

const props = defineProps<{
  events: TimelineEvent[]
}>()

const { t } = useI18n()

function eventIcon(type: string): string {
  const icons: Record<string, string> = {
    note: 'i-lucide-sticky-note',
    label: 'i-lucide-tag',
    conversation: 'i-lucide-message-square',
    attribute: 'i-lucide-sliders'
  }
  const fallback = 'i-lucide-circle-dot'
  return icons[type] ?? fallback
}

const timelineItems = computed<TimelineItem[]>(() =>
  props.events.map(event => ({
    date: new Date(event.createdAt).toLocaleString(),
    title: event.description,
    icon: eventIcon(event.type)
  }))
)
</script>

<template>
  <UPageCard variant="outline" :title="t('contacts.tabs.events')">
    <div v-if="!events.length" class="text-sm text-muted py-4">
      {{ t('common.noResults') }}
    </div>

    <UTimeline v-else :items="timelineItems" />
  </UPageCard>
</template>
