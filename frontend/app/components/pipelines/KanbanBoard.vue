<script setup lang="ts">
import { computed } from 'vue'
import KanbanColumn from './KanbanColumn.vue'
import { computeInsertPosition } from '~/utils/kanban'
import { usePipelineCardsStore, type PipelineCard } from '~/stores/pipelineCards'
import type { Pipeline, PipelineStage } from '~/stores/pipelines'

const props = defineProps<{
  pipeline: Pipeline
  canManage: boolean
}>()

const emit = defineEmits<{
  'add-card': [stageId: number]
  'edit-stage': [stage: PipelineStage]
  'delete-stage': [stage: PipelineStage]
  'open-card': [card: PipelineCard]
}>()

const cardsStore = usePipelineCardsStore()
const errorHandler = useErrorHandler()
const { t } = useI18n()

const stages = computed<PipelineStage[]>(() =>
  [...(props.pipeline.stages ?? [])].sort((a, b) => a.position - b.position)
)

function cardsFor(stageId: number | string): PipelineCard[] {
  return cardsStore.cardsInStage(props.pipeline.id, stageId)
}

function findStageIdFromTarget(el: HTMLElement | null): number | null {
  let cur: HTMLElement | null = el
  while (cur) {
    const v = cur.dataset?.stageId
    if (v) return Number(v)
    cur = cur.parentElement
  }
  return null
}

async function persistMove(card: PipelineCard, toStageId: number, position: number, originalStageId: number, originalPosition: number) {
  try {
    await cardsStore.move(props.pipeline.id, card.id, toStageId, position)
  } catch (err) {
    cardsStore.setMovedCard({
      cardId: card.id,
      pipelineId: props.pipeline.id,
      fromStageId: toStageId,
      toStageId: originalStageId,
      position: originalPosition
    })
    errorHandler.handle(err, { title: t('pipelines.card.moveFailed') })
  }
}

function handleDragEnd(evt: { item: HTMLElement, from: HTMLElement, to: HTMLElement, oldIndex?: number, newIndex?: number }) {
  const cardId = evt.item?.dataset?.cardId
  if (!cardId) return
  const fromStageId = findStageIdFromTarget(evt.from)
  const toStageId = findStageIdFromTarget(evt.to)
  if (fromStageId === null || toStageId === null) return
  const newIndex = evt.newIndex ?? 0

  const card = cardsStore.cardById(props.pipeline.id, cardId)
  if (!card) return

  const originalStageId = card.stageId
  const originalPosition = card.position

  const destList = cardsStore.cardsInStage(props.pipeline.id, toStageId)
  const newPosition = computeInsertPosition(destList, newIndex, cardId)

  if (originalStageId === toStageId && originalPosition === newPosition) return

  cardsStore.setMovedCard({
    cardId,
    pipelineId: props.pipeline.id,
    fromStageId: originalStageId,
    toStageId,
    position: newPosition
  })

  void persistMove(card, toStageId, newPosition, originalStageId, originalPosition)
}

function onAddCard(stageId: number) {
  emit('add-card', stageId)
}
function onEditStage(stage: PipelineStage) {
  emit('edit-stage', stage)
}
function onDeleteStage(stage: PipelineStage) {
  emit('delete-stage', stage)
}
function onOpenCard(card: PipelineCard) {
  emit('open-card', card)
}
</script>

<template>
  <div class="flex-1 min-w-0 flex gap-4 overflow-x-auto p-4">
    <KanbanColumn
      v-for="stage in stages"
      :key="stage.id"
      :stage="stage"
      :cards="cardsFor(stage.id)"
      :can-manage="canManage"
      :on-drag-end="handleDragEnd"
      @add-card="onAddCard"
      @edit-stage="onEditStage"
      @delete-stage="onDeleteStage"
      @open-card="onOpenCard"
    />

    <slot name="footer" />
  </div>
</template>
