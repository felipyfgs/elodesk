<script setup lang="ts">
import { computed } from 'vue'
import { useAgentsStore } from '~/stores/agents'
import { useLabelsStore } from '~/stores/labels'
import type { PipelineCard } from '~/stores/pipelineCards'

const props = defineProps<{
  card: PipelineCard
}>()

const emit = defineEmits<{
  open: [card: PipelineCard]
}>()

const { t } = useI18n()
const agentsStore = useAgentsStore()
const labelsStore = useLabelsStore()

const assignees = computed(() =>
  props.card.assigneeUserIds
    .map(id => agentsStore.items.find(a => String(a.userId) === String(id)))
    .filter((a): a is NonNullable<typeof a> => Boolean(a))
)

const labels = computed(() =>
  props.card.labelIds
    .map(id => labelsStore.list.find(l => String(l.id) === String(id)))
    .filter((l): l is NonNullable<typeof l> => Boolean(l))
)

const dueDateNear = computed(() => {
  if (!props.card.dueDate) return false
  const due = new Date(props.card.dueDate).getTime()
  const now = Date.now()
  const week = 7 * 24 * 60 * 60 * 1000
  return due - now < week && due - now > -week * 8
})

const valueLabel = computed(() => {
  if (props.card.valueCents === null || props.card.valueCents === undefined) return null
  const value = (props.card.valueCents) / 100
  const currency = props.card.valueCurrency || 'BRL'
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(value)
  } catch {
    return `${currency} ${value.toFixed(2)}`
  }
})

const dueLabel = computed(() => {
  if (!props.card.dueDate) return null
  try {
    return new Date(props.card.dueDate).toLocaleDateString()
  } catch {
    return props.card.dueDate
  }
})
</script>

<template>
  <UCard
    class="cursor-grab hover:shadow-md transition-shadow active:cursor-grabbing"
    :ui="{ body: 'p-3 sm:p-3' }"
    @click="emit('open', card)"
  >
    <div class="flex flex-col gap-2">
      <p class="text-sm font-medium text-default line-clamp-2">
        {{ card.title }}
      </p>

      <p
        v-if="card.description"
        class="text-xs text-muted line-clamp-2"
      >
        {{ card.description }}
      </p>

      <div v-if="labels.length" class="flex flex-wrap gap-1">
        <UBadge
          v-for="label in labels"
          :key="label.id"
          size="sm"
          :style="{ backgroundColor: label.color, color: '#fff' }"
        >
          {{ label.title }}
        </UBadge>
      </div>

      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2 min-w-0">
          <UChip
            v-if="dueLabel"
            :color="dueDateNear ? 'warning' : 'neutral'"
            inset
          >
            <span class="text-xs text-muted flex items-center gap-1">
              <UIcon name="i-lucide-calendar" class="size-3.5" />
              {{ dueLabel }}
            </span>
          </UChip>
          <span v-if="valueLabel" class="text-xs font-medium text-primary">
            {{ valueLabel }}
          </span>
        </div>

        <UAvatarGroup
          v-if="assignees.length"
          :max="3"
          size="xs"
          class="shrink-0"
        >
          <UAvatar
            v-for="agent in assignees"
            :key="agent.id"
            :alt="agent.name"
            :title="agent.name"
            size="xs"
          />
        </UAvatarGroup>
      </div>

      <p
        v-if="card.linkedEntityType"
        class="text-xs text-muted flex items-center gap-1"
      >
        <UIcon
          :name="card.linkedEntityType === 'contact' ? 'i-lucide-user' : 'i-lucide-message-square'"
          class="size-3.5"
        />
        {{ t(`pipelines.card.linked${card.linkedEntityType === 'contact' ? 'Contact' : 'Conversation'}`) }}
      </p>
    </div>
  </UCard>
</template>
