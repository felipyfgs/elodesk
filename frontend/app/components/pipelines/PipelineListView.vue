<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { computed, h, resolveComponent } from 'vue'
import { useAgentsStore } from '~/stores/agents'
import { useLabelsStore } from '~/stores/labels'
import type { PipelineCard } from '~/stores/pipelineCards'
import type { Pipeline, PipelineStage } from '~/stores/pipelines'

const props = defineProps<{
  pipeline: Pipeline
  cards: PipelineCard[]
}>()

const emit = defineEmits<{
  'open-card': [card: PipelineCard]
}>()

const { t } = useI18n()
const agentsStore = useAgentsStore()
const labelsStore = useLabelsStore()

const stagesById = computed<Record<string, PipelineStage>>(() => {
  const out: Record<string, PipelineStage> = {}
  for (const s of props.pipeline.stages ?? []) out[String(s.id)] = s
  return out
})

const columns: TableColumn<PipelineCard>[] = [
  {
    accessorKey: 'title',
    header: () => t('pipelines.list.title'),
    enableSorting: true
  },
  {
    accessorKey: 'stageId',
    header: () => t('pipelines.list.stage'),
    enableSorting: true,
    sortingFn: (a, b) => a.original.position - b.original.position
  },
  {
    accessorKey: 'assigneeUserIds',
    header: () => t('pipelines.list.assignees'),
    enableSorting: false
  },
  {
    accessorKey: 'labelIds',
    header: () => t('pipelines.list.labels'),
    enableSorting: false
  },
  {
    accessorKey: 'dueDate',
    header: () => t('pipelines.list.dueDate'),
    enableSorting: true
  },
  {
    accessorKey: 'valueCents',
    header: () => t('pipelines.list.value'),
    enableSorting: true
  },
  {
    accessorKey: 'createdAt',
    header: () => t('pipelines.list.createdAt'),
    enableSorting: true
  }
]

function formatValue(card: PipelineCard): string | null {
  if (card.valueCents === null || card.valueCents === undefined) return null
  const value = card.valueCents / 100
  const currency = card.valueCurrency || 'BRL'
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(value)
  } catch {
    return `${currency} ${value.toFixed(2)}`
  }
}

function formatDate(value?: string | null): string {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleDateString()
  } catch {
    return value
  }
}

function stageFor(stageId: number | string): PipelineStage | undefined {
  return stagesById.value[String(stageId)]
}

function agentName(userId: number | string): string {
  const a = agentsStore.items.find(x => String(x.userId) === String(userId))
  return a?.name || a?.email || `#${userId}`
}

const UBadge = resolveComponent('UBadge')
const UAvatarGroup = resolveComponent('UAvatarGroup')
const UAvatar = resolveComponent('UAvatar')

function renderAvatars(card: PipelineCard) {
  if (!card.assigneeUserIds.length) return h('span', { class: 'text-xs text-muted' }, '—')
  return h(
    UAvatarGroup,
    { size: 'xs', max: 3 },
    () => card.assigneeUserIds.map(id => h(UAvatar, {
      key: id,
      alt: agentName(id),
      title: agentName(id),
      size: 'xs'
    }))
  )
}

function renderLabels(card: PipelineCard) {
  if (!card.labelIds.length) return h('span', { class: 'text-xs text-muted' }, '—')
  const items = card.labelIds.map((id) => {
    const label = labelsStore.list.find(l => String(l.id) === String(id))
    if (!label) return null
    return h(
      UBadge,
      {
        key: id,
        size: 'sm',
        style: { backgroundColor: label.color, color: '#fff' }
      },
      () => label.title
    )
  }).filter(Boolean)
  return h('div', { class: 'flex flex-wrap gap-1' }, items)
}

function renderStage(card: PipelineCard) {
  const stage = stageFor(card.stageId)
  if (!stage) return h('span', { class: 'text-xs text-muted' }, '—')
  return h(
    UBadge,
    {
      size: 'sm',
      variant: 'subtle',
      style: { color: stage.color, borderColor: stage.color }
    },
    () => stage.name
  )
}

function onRowClick(card: PipelineCard) {
  emit('open-card', card)
}
</script>

<template>
  <div class="max-w-7xl mx-auto w-full p-4">
    <UTable
      :data="cards"
      :columns="columns"
      :empty-state="{ icon: 'i-lucide-kanban-square', label: t('pipelines.list.empty') }"
      :ui="{
        base: 'table-fixed border-separate border-spacing-0',
        thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
        th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
        td: 'border-b border-default',
        tr: 'cursor-pointer hover:bg-elevated/30',
        separator: 'h-0'
      }"
    >
      <template #title-cell="{ row }">
        <button
          type="button"
          class="font-medium text-default hover:text-primary text-left truncate w-full"
          @click="onRowClick(row.original)"
        >
          {{ row.original.title || t('pipelines.card.untitled') }}
        </button>
      </template>

      <template #stageId-cell="{ row }">
        <component :is="renderStage(row.original)" />
      </template>

      <template #assigneeUserIds-cell="{ row }">
        <component :is="renderAvatars(row.original)" />
      </template>

      <template #labelIds-cell="{ row }">
        <component :is="renderLabels(row.original)" />
      </template>

      <template #dueDate-cell="{ row }">
        <span class="text-sm text-muted">{{ formatDate(row.original.dueDate) }}</span>
      </template>

      <template #valueCents-cell="{ row }">
        <span class="text-sm font-medium text-primary">
          {{ formatValue(row.original) ?? '—' }}
        </span>
      </template>

      <template #createdAt-cell="{ row }">
        <span class="text-sm text-muted">{{ formatDate(row.original.createdAt) }}</span>
      </template>
    </UTable>
  </div>
</template>
