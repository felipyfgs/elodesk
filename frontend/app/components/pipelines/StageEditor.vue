<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { reactive, ref, watch } from 'vue'
import { createStageSchema, type CreateStageForm, type TerminalKind } from '~/schemas/pipeline'
import { usePipelinesStore, type PipelineStage } from '~/stores/pipelines'

const props = defineProps<{
  pipelineId: number | string
  stage?: PipelineStage | null
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'saved': [stage: PipelineStage]
}>()

const { t } = useI18n()
const pipelinesStore = usePipelinesStore()
const errorHandler = useErrorHandler()
const isLoading = ref(false)
const formRef = useTemplateRef('formRef')

const initialState = (): CreateStageForm => ({
  name: '',
  color: '#94a3b8',
  is_terminal: false,
  terminal_kind: undefined
})

const form = reactive<CreateStageForm>(initialState())

watch(() => [props.open, props.stage], () => {
  if (props.open && props.stage) {
    form.name = props.stage.name
    form.color = props.stage.color
    form.is_terminal = props.stage.isTerminal
    form.terminal_kind = (props.stage.terminalKind ?? undefined) as TerminalKind | undefined
  } else if (props.open) {
    Object.assign(form, initialState())
  }
}, { immediate: true })

const TERMINAL_OPTIONS: { value: TerminalKind, label: string }[] = [
  { value: 'won', label: 'won' },
  { value: 'lost', label: 'lost' },
  { value: 'resolved', label: 'resolved' }
]

async function submit(event: FormSubmitEvent<CreateStageForm>) {
  isLoading.value = true
  try {
    const body = {
      name: event.data.name,
      color: event.data.color,
      is_terminal: event.data.is_terminal ?? false,
      terminal_kind: event.data.is_terminal ? event.data.terminal_kind : undefined
    }
    let saved: PipelineStage
    if (props.stage) {
      saved = await pipelinesStore.updateStage(props.pipelineId, props.stage.id, body)
    } else {
      saved = await pipelinesStore.addStage(props.pipelineId, body)
    }
    emit('saved', saved)
    emit('update:open', false)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.stage.saveFailed') })
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <UModal
    :open="props.open"
    :title="props.stage ? t('pipelines.stage.edit') : t('pipelines.stage.create')"
    :ui="{ content: 'sm:max-w-md' }"
    @update:open="value => emit('update:open', value)"
  >
    <template #body>
      <UForm
        ref="formRef"
        :schema="createStageSchema"
        :state="form"
        class="flex flex-col gap-4"
        @submit="submit"
      >
        <UFormField :label="t('pipelines.stage.name')" name="name" required>
          <UInput v-model="form.name" autofocus class="w-full" />
        </UFormField>

        <UFormField :label="t('pipelines.stage.color')" name="color">
          <div class="flex gap-2 items-center w-full">
            <UPopover :content="{ side: 'bottom', align: 'start' }">
              <UButton
                type="button"
                color="neutral"
                variant="outline"
                class="shrink-0"
              >
                <span
                  class="size-4 rounded-full ring ring-default"
                  :style="{ backgroundColor: form.color || '#94a3b8' }"
                />
              </UButton>
              <template #content>
                <UColorPicker v-model="form.color" class="p-2" />
              </template>
            </UPopover>
            <UInput v-model="form.color" class="flex-1" />
          </div>
        </UFormField>

        <UFormField name="is_terminal">
          <UCheckbox v-model="form.is_terminal" :label="t('pipelines.stage.isTerminal')" />
        </UFormField>

        <UFormField
          v-if="form.is_terminal"
          :label="t('pipelines.stage.terminalKind')"
          name="terminal_kind"
        >
          <USelect
            v-model="form.terminal_kind"
            :items="TERMINAL_OPTIONS"
            value-key="value"
            label-key="label"
            class="w-full"
          />
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="isLoading"
          @click="emit('update:open', false)"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton :loading="isLoading" @click="formRef?.submit()">
          {{ t('common.save') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
