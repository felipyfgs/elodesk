<script setup lang="ts">
import type { FilterConditionForm } from '~/schemas/savedFilter'

const props = defineProps<{
  modelValue: boolean
  filterType: 'conversation' | 'contact'
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: { name: string; filter_type: string; query: { operator: string; conditions: FilterConditionForm[] } }]
}>()

const { t } = useI18n()
const api = useApi()

const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const saving = ref(false)

const form = reactive({
  name: '',
  conditions: [] as FilterConditionForm[]
})

const operators = [
  { value: 'equal_to', label: 'Igual a' },
  { value: 'not_equal_to', label: 'Diferente de' },
  { value: 'contains', label: 'Contém' },
  { value: 'starts_with', label: 'Começa com' },
  { value: 'greater_than', label: 'Maior que' },
  { value: 'less_than', label: 'Menor que' },
  { value: 'in', label: 'Está em' },
  { value: 'between', label: 'Entre' },
  { value: 'is_null', label: 'É nulo' },
  { value: 'is_not_null', label: 'Não é nulo' }
]

function addCondition() {
  form.conditions.push({
    attribute_key: '',
    filter_operator: 'equal_to',
    value: null
  })
}

function removeCondition(index: number) {
  form.conditions.splice(index, 1)
}

watch(isOpen, (v) => {
  if (!v) {
    form.name = ''
    form.conditions = []
  }
})

async function save() {
  if (!form.name || !form.conditions.length) return
  saving.value = true
  try {
    const payload = {
      name: form.name,
      filter_type: props.filterType,
      query: { operator: 'and', conditions: form.conditions }
    }
    await api('/saved-filters', { method: 'POST', body: payload })
    emit('save', payload)
    isOpen.value = false
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="isOpen" :title="t('savedFilters.create')">
    <template #body>
      <div class="space-y-4">
        <UFormField :label="t('savedFilters.name')">
          <UInput v-model="form.name" class="w-full" />
        </UFormField>

        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium">{{ t('savedFilters.title') }}</span>
            <UButton
              icon="i-lucide-plus"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="addCondition"
            />
          </div>

          <div
            v-for="(cond, idx) in form.conditions"
            :key="idx"
            class="flex items-center gap-2"
          >
            <UInput
              v-model="cond.attribute_key"
              placeholder="campo"
              class="flex-1"
            />
            <USelect
              v-model="cond.filter_operator"
              :items="operators"
              class="w-40"
            />
            <UInput
              v-model="cond.value"
              placeholder="valor"
              class="flex-1"
              :disabled="cond.filter_operator === 'is_null' || cond.filter_operator === 'is_not_null'"
            />
            <UButton
              icon="i-lucide-x"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="removeCondition(idx)"
            />
          </div>

          <p v-if="!form.conditions.length" class="text-sm text-muted">
            {{ t('savedFilters.empty') }}
          </p>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" @click="isOpen = false">
          {{ t('common.cancel') }}
        </UButton>
        <UButton :loading="saving" :disabled="!form.name || !form.conditions.length" @click="save">
          {{ t('common.save') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
