<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { usePipelinesStore, type PipelineTemplate, type Pipeline } from '~/stores/pipelines'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'created': [pipeline: Pipeline]
}>()

const { t } = useI18n()
const pipelinesStore = usePipelinesStore()
const errorHandler = useErrorHandler()

const selectedKey = ref<string | null>(null)
const name = ref('')
const isLoading = ref(false)

const templates = computed<PipelineTemplate[]>(() => pipelinesStore.templates)
const selected = computed(() => templates.value.find(tpl => tpl.key === selectedKey.value) ?? null)

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    if (!templates.value.length) {
      try {
        await pipelinesStore.fetchTemplates()
      } catch (err) {
        errorHandler.handle(err, { title: t('pipelines.templates.fetchFailed') })
      }
    }
    selectedKey.value = null
    name.value = ''
  }
})

function pick(template: PipelineTemplate) {
  selectedKey.value = template.key
  if (!name.value) name.value = template.name
}

async function create() {
  if (!selectedKey.value || !name.value.trim()) return
  isLoading.value = true
  try {
    const created = await pipelinesStore.create({
      name: name.value.trim(),
      template_key: selectedKey.value
    })
    emit('created', created)
    emit('update:open', false)
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.createFailed') })
  } finally {
    isLoading.value = false
  }
}

const ICON_FOR_KEY: Record<string, string> = {
  'sales-crm': 'i-lucide-trending-up',
  'support': 'i-lucide-life-buoy',
  'tasks': 'i-lucide-list-checks',
  'blank': 'i-lucide-plus-square'
}
</script>

<template>
  <UModal
    :open="props.open"
    :title="t('pipelines.templates.title')"
    :description="t('pipelines.templates.subtitle')"
    :ui="{ content: 'sm:max-w-3xl' }"
    @update:open="value => emit('update:open', value)"
  >
    <template #body>
      <div class="flex flex-col gap-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <UCard
            v-for="tpl in templates"
            :key="tpl.key"
            class="cursor-pointer transition-all"
            :class="selectedKey === tpl.key
              ? 'ring-2 ring-primary'
              : 'hover:bg-elevated/50'"
            :ui="{ body: 'p-4' }"
            @click="pick(tpl)"
          >
            <div class="flex items-start gap-3">
              <UIcon
                :name="ICON_FOR_KEY[tpl.key] ?? 'i-lucide-kanban-square'"
                class="size-6 text-primary shrink-0"
              />
              <div class="min-w-0">
                <h4 class="text-sm font-semibold text-default">
                  {{ tpl.name }}
                </h4>
                <p class="text-xs text-muted mt-1 line-clamp-2">
                  {{ tpl.description }}
                </p>
                <div class="flex flex-wrap gap-1 mt-2">
                  <UBadge
                    v-for="stage in tpl.stages"
                    :key="stage.name"
                    size="sm"
                    variant="subtle"
                    color="neutral"
                  >
                    {{ stage.name }}
                  </UBadge>
                </div>
              </div>
            </div>
          </UCard>
        </div>

        <UFormField v-if="selected" :label="t('pipelines.name')" required>
          <UInput
            v-model="name"
            :placeholder="selected.name"
            class="w-full"
          />
        </UFormField>
      </div>
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
        <UButton
          :loading="isLoading"
          :disabled="!selectedKey || !name.trim()"
          @click="create"
        >
          {{ t('pipelines.create') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
