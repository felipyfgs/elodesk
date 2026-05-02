<script setup lang="ts">
import { ConfirmModal } from '#components'
import { computed, onMounted, ref, watch } from 'vue'
import { useAgentsStore } from '~/stores/agents'
import { useLabelsStore } from '~/stores/labels'
import { usePipelineCardsStore, type PipelineCard } from '~/stores/pipelineCards'
import { usePipelinesStore, type Pipeline, type PipelineStage } from '~/stores/pipelines'

definePageMeta({ layout: 'dashboard' })

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const pipelinesStore = usePipelinesStore()
const cardsStore = usePipelineCardsStore()
const agentsStore = useAgentsStore()
const labelsStore = useLabelsStore()
const errorHandler = useErrorHandler()
const confirmOverlay = useOverlay().create(ConfirmModal)

const aid = computed(() => route.params.accountId as string)
const pipelineId = computed(() => route.params.id as string)

const pipeline = computed<Pipeline | undefined>(() => pipelinesStore.byId(pipelineId.value))

type ViewMode = 'kanban' | 'list'

const viewMode = computed<ViewMode>(() => {
  const v = route.query.view
  return v === 'list' ? 'list' : 'kanban'
})

function setViewMode(mode: ViewMode) {
  void router.replace({ query: { ...route.query, view: mode } })
}

const allCards = computed(() => {
  const stages = pipeline.value?.stages ?? []
  const out = []
  for (const stage of stages) {
    out.push(...cardsStore.cardsInStage(pipelineId.value, stage.id))
  }
  return out
})

const viewItems = computed(() => [
  { label: t('pipelines.view.kanban'), icon: 'i-lucide-kanban-square', value: 'kanban' as ViewMode },
  { label: t('pipelines.view.list'), icon: 'i-lucide-list', value: 'list' as ViewMode }
])

const fetching = ref(false)
const stageEditorOpen = ref(false)
const editingStage = ref<PipelineStage | null>(null)
const cardSlideoverOpen = ref(false)
const activeCard = ref<PipelineCard | null>(null)
const renameOpen = ref(false)
const renameValue = ref('')

async function load() {
  fetching.value = true
  try {
    await Promise.all([
      pipelinesStore.fetchOne(pipelineId.value),
      cardsStore.fetchByPipeline(pipelineId.value),
      agentsStore.items.length ? Promise.resolve() : agentsStore.fetch(),
      labelsStore.list.length ? Promise.resolve() : Promise.resolve()
    ])
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.fetchFailed'), onRetry: load })
  } finally {
    fetching.value = false
  }
}

onMounted(load)
watch(pipelineId, load)

const linkPickerOpen = ref(false)
const linkPickerStageId = ref<number | null>(null)

function openAddCard(stageId: number) {
  const kind = pipeline.value?.cardKind
  if (kind === 'contact' || kind === 'conversation') {
    linkPickerStageId.value = stageId
    linkPickerOpen.value = true
    return
  }
  void createCardQuick(stageId)
}

async function createCardQuick(
  stageId: number,
  link?: { type: 'contact' | 'conversation', id: number, label: string }
) {
  try {
    const card = await cardsStore.create(pipelineId.value, {
      stage_id: stageId,
      title: link?.label ?? t('pipelines.card.untitled'),
      ...(link ? { linked_entity_type: link.type, linked_entity_id: link.id } : {})
    })
    activeCard.value = card
    cardSlideoverOpen.value = true
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.card.createFailed') })
  }
}

function onLinkedEntityPicked(payload: { type: 'contact' | 'conversation', id: number, label: string }) {
  if (linkPickerStageId.value === null) return
  void createCardQuick(linkPickerStageId.value, payload)
  linkPickerStageId.value = null
}

function openEditStage(stage: PipelineStage) {
  editingStage.value = stage
  stageEditorOpen.value = true
}

function openCreateStage() {
  editingStage.value = null
  stageEditorOpen.value = true
}

async function deleteStage(stage: PipelineStage) {
  const ok = await confirmOverlay.open({
    title: t('pipelines.stage.delete'),
    description: t('pipelines.stage.deleteConfirm'),
    confirmLabel: t('common.delete'),
    itemName: stage.name
  })
  if (!ok) return
  try {
    await pipelinesStore.deleteStage(pipelineId.value, stage.id)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.stage.deleteFailed') })
  }
}

function openCard(card: PipelineCard) {
  activeCard.value = card
  cardSlideoverOpen.value = true
}

function openRename() {
  if (!pipeline.value) return
  renameValue.value = pipeline.value.name
  renameOpen.value = true
}

async function rename() {
  if (!pipeline.value || !renameValue.value.trim()) return
  try {
    await pipelinesStore.update(pipeline.value.id, { name: renameValue.value.trim() })
    renameOpen.value = false
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.updateFailed') })
  }
}

async function archive() {
  if (!pipeline.value) return
  const ok = await confirmOverlay.open({
    title: t('pipelines.archive'),
    description: t('pipelines.archiveConfirm'),
    confirmLabel: t('pipelines.archive'),
    itemName: pipeline.value.name
  })
  if (!ok) return
  try {
    await pipelinesStore.archive(pipeline.value.id)
    void navigateTo(`/accounts/${aid.value}/pipelines`)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.archiveFailed') })
  }
}

const headerMenu = computed(() => [
  [{
    label: t('common.edit'),
    icon: 'i-lucide-pencil',
    onSelect: openRename
  }, {
    label: t('pipelines.stage.create'),
    icon: 'i-lucide-plus',
    onSelect: openCreateStage
  }, {
    label: t('pipelines.archive'),
    icon: 'i-lucide-archive',
    onSelect: archive
  }]
])

watch(activeCard, (card) => {
  if (!cardSlideoverOpen.value) return
  if (!card) return
  const fresh = cardsStore.cardById(pipelineId.value, card.id)
  if (fresh && fresh !== card) activeCard.value = fresh
})
</script>

<template>
  <UDashboardPanel id="pipeline-board">
    <template #header>
      <UDashboardNavbar :title="pipeline?.name ?? t('pipelines.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <div class="flex gap-1 rounded-md bg-elevated/40 p-1">
            <UButton
              v-for="item in viewItems"
              :key="item.value"
              :icon="item.icon"
              :color="viewMode === item.value ? 'primary' : 'neutral'"
              :variant="viewMode === item.value ? 'solid' : 'ghost'"
              size="sm"
              @click="setViewMode(item.value)"
            >
              {{ item.label }}
            </UButton>
          </div>
          <UButton
            variant="ghost"
            icon="i-lucide-plus"
            @click="openCreateStage"
          >
            {{ t('pipelines.stage.create') }}
          </UButton>
          <UDropdownMenu :items="headerMenu">
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-ellipsis-vertical"
              :aria-label="t('common.more')"
            />
          </UDropdownMenu>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="fetching && !pipeline" class="flex items-center justify-center h-full">
        <USkeleton class="h-64 w-full" />
      </div>

      <div v-else-if="!pipeline" class="flex flex-col items-center text-center gap-2 py-12">
        <UIcon name="i-lucide-kanban-square" class="size-10 text-muted" />
        <p class="text-sm text-muted">
          {{ t('pipelines.notFound') }}
        </p>
      </div>

      <PipelinesKanbanBoard
        v-else-if="viewMode === 'kanban'"
        :pipeline="pipeline"
        :can-manage="true"
        @add-card="openAddCard"
        @edit-stage="openEditStage"
        @delete-stage="deleteStage"
        @open-card="openCard"
      />

      <PipelinesPipelineListView
        v-else
        :pipeline="pipeline"
        :cards="allCards"
        @open-card="openCard"
      />

      <PipelinesStageEditor
        v-if="pipeline"
        v-model:open="stageEditorOpen"
        :pipeline-id="pipeline.id"
        :stage="editingStage"
      />

      <PipelinesCardSlideover
        v-model:open="cardSlideoverOpen"
        :card="activeCard"
        :pipeline-id="pipelineId"
      />

      <PipelinesLinkedEntityPicker
        v-if="pipeline && (pipeline.cardKind === 'contact' || pipeline.cardKind === 'conversation')"
        v-model:open="linkPickerOpen"
        :kind="pipeline.cardKind"
        @picked="onLinkedEntityPicked"
      />

      <UModal v-model:open="renameOpen" :title="t('pipelines.edit')" :ui="{ content: 'sm:max-w-md' }">
        <template #body>
          <UFormField :label="t('pipelines.name')">
            <UInput v-model="renameValue" autofocus class="w-full" />
          </UFormField>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2 w-full">
            <UButton color="neutral" variant="ghost" @click="renameOpen = false">
              {{ t('common.cancel') }}
            </UButton>
            <UButton :disabled="!renameValue.trim()" @click="rename">
              {{ t('common.save') }}
            </UButton>
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
