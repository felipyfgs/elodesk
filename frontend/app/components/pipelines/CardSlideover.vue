<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useAgentsStore } from '~/stores/agents'
import { useLabelsStore } from '~/stores/labels'
import { usePipelineCardsStore, type PipelineCard } from '~/stores/pipelineCards'

const props = defineProps<{
  open: boolean
  card: PipelineCard | null
  pipelineId: number | string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const { t } = useI18n()
const cardsStore = usePipelineCardsStore()
const agentsStore = useAgentsStore()
const labelsStore = useLabelsStore()
const errorHandler = useErrorHandler()

const isSaving = ref(false)
const form = reactive<{
  title: string
  description: string
  dueDate: string
  valueAmount: string
  valueCurrency: string
}>({
  title: '',
  description: '',
  dueDate: '',
  valueAmount: '',
  valueCurrency: 'BRL'
})

watch(() => [props.open, props.card?.id], () => {
  if (props.open && props.card) {
    form.title = props.card.title
    form.description = props.card.description ?? ''
    form.dueDate = props.card.dueDate ?? ''
    form.valueAmount = props.card.valueCents !== null && props.card.valueCents !== undefined
      ? String((props.card.valueCents) / 100)
      : ''
    form.valueCurrency = props.card.valueCurrency ?? 'BRL'
  }
}, { immediate: true })

const assigneeOptions = computed(() => agentsStore.items.map(a => ({
  label: a.name || a.email,
  value: a.userId
})))

const labelOptions = computed(() => labelsStore.list.map(l => ({
  label: l.title,
  value: Number(l.id),
  color: l.color
})))

const selectedAssignees = computed({
  get: () => props.card?.assigneeUserIds ?? [],
  set: () => { /* handled via add/remove */ }
})

const selectedLabels = computed({
  get: () => props.card?.labelIds ?? [],
  set: () => { /* handled via apply/remove */ }
})

async function save() {
  if (!props.card) return
  isSaving.value = true
  try {
    const valueCents = form.valueAmount.trim()
      ? Math.round(parseFloat(form.valueAmount) * 100)
      : null
    await cardsStore.update(props.pipelineId, props.card.id, {
      title: form.title.trim(),
      description: form.description.trim() ? form.description : null,
      due_date: form.dueDate || null,
      value_cents: Number.isFinite(valueCents) ? valueCents : null,
      value_currency: form.valueCurrency || null
    })
    emit('update:open', false)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.card.saveFailed') })
  } finally {
    isSaving.value = false
  }
}

async function toggleAssignee(userId: number) {
  if (!props.card) return
  const already = props.card.assigneeUserIds.some(id => Number(id) === userId)
  try {
    if (already) await cardsStore.removeAssignee(props.pipelineId, props.card.id, userId)
    else await cardsStore.addAssignee(props.pipelineId, props.card.id, userId)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.card.saveFailed') })
  }
}

async function toggleLabel(labelId: number) {
  if (!props.card) return
  const already = props.card.labelIds.some(id => Number(id) === labelId)
  try {
    if (already) await cardsStore.removeLabel(props.pipelineId, props.card.id, labelId)
    else await cardsStore.applyLabel(props.pipelineId, props.card.id, labelId)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.card.saveFailed') })
  }
}

async function deleteCard() {
  if (!props.card) return
  try {
    await cardsStore.delete(props.pipelineId, props.card.id)
    emit('update:open', false)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.card.deleteFailed') })
  }
}
</script>

<template>
  <USlideover :open="props.open" @update:open="value => emit('update:open', value)">
    <template #content>
      <div v-if="card" class="p-6 flex flex-col gap-4">
        <div class="flex items-center justify-between gap-2">
          <h3 class="text-lg font-semibold text-default flex-1 truncate">
            {{ card.title || t('pipelines.card.untitled') }}
          </h3>
          <UButton
            color="error"
            variant="ghost"
            icon="i-lucide-trash-2"
            :aria-label="t('common.delete')"
            @click="deleteCard"
          />
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('common.close')"
            @click="emit('update:open', false)"
          />
        </div>

        <UFormField :label="t('pipelines.card.title')">
          <UInput v-model="form.title" class="w-full" />
        </UFormField>

        <UFormField :label="t('pipelines.card.description')">
          <UTextarea
            v-model="form.description"
            :rows="4"
            autoresize
            class="w-full"
          />
        </UFormField>

        <div class="grid grid-cols-2 gap-3">
          <UFormField :label="t('pipelines.card.dueDate')">
            <UInput v-model="form.dueDate" type="date" class="w-full" />
          </UFormField>
          <UFormField :label="t('pipelines.card.value')">
            <UInput
              v-model="form.valueAmount"
              type="number"
              step="0.01"
              min="0"
              class="w-full"
            />
          </UFormField>
        </div>

        <UFormField :label="t('pipelines.card.assignees')">
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="opt in assigneeOptions"
              :key="opt.value"
              size="sm"
              :variant="selectedAssignees.some(id => Number(id) === opt.value) ? 'solid' : 'outline'"
              color="primary"
              @click="toggleAssignee(opt.value)"
            >
              {{ opt.label }}
            </UButton>
            <p v-if="!assigneeOptions.length" class="text-xs text-muted">
              {{ t('pipelines.card.noAgents') }}
            </p>
          </div>
        </UFormField>

        <UFormField :label="t('pipelines.card.labels')">
          <div class="flex flex-wrap gap-2">
            <UBadge
              v-for="opt in labelOptions"
              :key="opt.value"
              :variant="selectedLabels.some(id => Number(id) === opt.value) ? 'solid' : 'outline'"
              :style="selectedLabels.some(id => Number(id) === opt.value)
                ? { backgroundColor: opt.color, color: '#fff' }
                : { borderColor: opt.color, color: opt.color }"
              class="cursor-pointer"
              @click="toggleLabel(opt.value)"
            >
              {{ opt.label }}
            </UBadge>
            <p v-if="!labelOptions.length" class="text-xs text-muted">
              {{ t('pipelines.card.noLabels') }}
            </p>
          </div>
        </UFormField>

        <div
          v-if="card.linkedEntityType && card.linkedEntityId"
          class="rounded-lg border border-default p-3 bg-elevated/30"
        >
          <p class="text-xs text-muted mb-1">
            {{ t(`pipelines.card.linked${card.linkedEntityType === 'contact' ? 'Contact' : 'Conversation'}`) }}
          </p>
          <NuxtLink
            v-if="card.linkedEntityType === 'contact'"
            :to="`/contacts/${card.linkedEntityId}`"
            class="text-sm text-primary hover:underline flex items-center gap-1"
          >
            <UIcon name="i-lucide-user" class="size-4" />
            #{{ card.linkedEntityId }}
          </NuxtLink>
          <NuxtLink
            v-else-if="card.linkedEntityType === 'conversation'"
            :to="`/conversations/${card.linkedEntityId}`"
            class="text-sm text-primary hover:underline flex items-center gap-1"
          >
            <UIcon name="i-lucide-message-square" class="size-4" />
            #{{ card.linkedEntityId }}
          </NuxtLink>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <UButton
            variant="ghost"
            color="neutral"
            :disabled="isSaving"
            @click="emit('update:open', false)"
          >
            {{ t('common.cancel') }}
          </UButton>
          <UButton :loading="isSaving" @click="save">
            {{ t('common.save') }}
          </UButton>
        </div>
      </div>
    </template>
  </USlideover>
</template>
