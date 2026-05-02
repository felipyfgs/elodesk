<script setup lang="ts">
import { computed } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import KanbanCard from './KanbanCard.vue'
import type { PipelineStage } from '~/stores/pipelines'
import type { PipelineCard } from '~/stores/pipelineCards'

const props = defineProps<{
  stage: PipelineStage
  cards: PipelineCard[]
  canManage: boolean
  onDragEnd: (evt: { item: HTMLElement, from: HTMLElement, to: HTMLElement, oldIndex?: number, newIndex?: number }) => void
}>()

const emit = defineEmits<{
  'add-card': [stageId: number]
  'edit-stage': [stage: PipelineStage]
  'delete-stage': [stage: PipelineStage]
  'open-card': [card: PipelineCard]
}>()

const { t } = useI18n()

const localCards = computed({
  get: () => props.cards,
  set: () => { /* parent owns the store; no-op */ }
})

const menuItems = computed(() => {
  if (!props.canManage) return []
  return [
    [{
      label: t('pipelines.stage.edit'),
      icon: 'i-lucide-pencil',
      onSelect: () => emit('edit-stage', props.stage)
    }, {
      label: t('pipelines.stage.delete'),
      icon: 'i-lucide-trash-2',
      onSelect: () => emit('delete-stage', props.stage)
    }]
  ]
})
</script>

<template>
  <UCard
    class="flex-shrink-0 w-72 sm:w-80 flex flex-col bg-elevated/40 max-h-[calc(100vh-12rem)]"
    :ui="{ header: 'p-3 sm:p-3', body: 'p-2 sm:p-2 flex-1 overflow-hidden flex flex-col gap-2' }"
  >
    <template #header>
      <div class="flex items-center gap-2 min-w-0">
        <span
          class="size-2.5 rounded-full shrink-0"
          :style="{ backgroundColor: stage.color }"
        />
        <h3 class="text-sm font-semibold text-default truncate flex-1 min-w-0">
          {{ stage.name }}
        </h3>
        <UBadge size="sm" variant="subtle" color="neutral">
          {{ cards.length }}
        </UBadge>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-lucide-plus"
          :aria-label="t('pipelines.card.create')"
          @click="emit('add-card', Number(stage.id))"
        />
        <UDropdownMenu v-if="menuItems.length" :items="menuItems">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-lucide-ellipsis-vertical"
            :aria-label="t('common.more')"
          />
        </UDropdownMenu>
      </div>
    </template>

    <UScrollArea class="flex-1">
      <VueDraggable
        v-model="localCards"
        :group="{ name: 'cards', pull: true, put: true }"
        :animation="150"
        ghost-class="opacity-30"
        :data-stage-id="stage.id"
        :on-end="onDragEnd"
        class="flex flex-col gap-2 min-h-[40px]"
      >
        <KanbanCard
          v-for="card in cards"
          :key="card.id"
          :data-card-id="card.id"
          :card="card"
          @open="emit('open-card', card)"
        />
      </VueDraggable>

      <UButton
        v-if="!cards.length"
        block
        variant="ghost"
        color="neutral"
        icon="i-lucide-plus"
        size="sm"
        class="mt-2"
        @click="emit('add-card', Number(stage.id))"
      >
        {{ t('pipelines.card.create') }}
      </UButton>
    </UScrollArea>
  </UCard>
</template>
