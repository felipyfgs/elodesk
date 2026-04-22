<script setup lang="ts">
import type { FilterConditionForm } from '~/schemas/savedFilter'
import type { SavedFilter } from '~/stores/savedFilters'
import { useAuthStore } from '~/stores/auth'
import { useSavedFiltersStore } from '~/stores/savedFilters'

export interface FilterQueryPayload {
  operator: 'AND' | 'OR'
  conditions: FilterConditionForm[]
}

const props = defineProps<{
  modelValue: boolean
  filterType: 'conversation' | 'contact'
  initialQuery?: FilterQueryPayload | null
  initialName?: string
  editingId?: string | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'apply': [payload: FilterQueryPayload]
  'save': [filter: SavedFilter]
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const savedFilters = useSavedFiltersStore()
const toast = useToast()
const { attributesFor } = useFilterAttributes()

const isOpen = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v)
})

const attributes = attributesFor(props.filterType)

function emptyCondition(): FilterConditionForm {
  return { attribute_key: '', filter_operator: 'equal_to', value: null }
}

const operator = ref<'AND' | 'OR'>('AND')
const conditions = ref<FilterConditionForm[]>([emptyCondition()])
const name = ref('')
const showSave = ref(false)
const saving = ref(false)
const applying = ref(false)

function resetForm() {
  operator.value = 'AND'
  conditions.value = [emptyCondition()]
  name.value = ''
  showSave.value = false
}

function hydrateFromInitial() {
  if (props.initialQuery && props.initialQuery.conditions?.length) {
    operator.value = props.initialQuery.operator
    conditions.value = props.initialQuery.conditions.map(c => ({ ...c }))
  } else {
    conditions.value = [emptyCondition()]
  }
  name.value = props.initialName ?? ''
  showSave.value = !!props.editingId
}

watch(isOpen, (v) => {
  if (v) hydrateFromInitial()
  else resetForm()
})

function addCondition() {
  conditions.value.push(emptyCondition())
}

function removeCondition(index: number) {
  conditions.value.splice(index, 1)
  if (conditions.value.length === 0) conditions.value.push(emptyCondition())
}

const connectorItems = computed(() => [
  { label: t('savedFilters.connector.and'), value: 'AND' },
  { label: t('savedFilters.connector.or'), value: 'OR' }
])

const validConditions = computed(() =>
  conditions.value.filter(c =>
    c.attribute_key
    && c.filter_operator
    && (
      c.filter_operator === 'is_null'
      || c.filter_operator === 'is_not_null'
      || (c.value !== null && c.value !== '' && !(Array.isArray(c.value) && c.value.length === 0))
    )
  )
)

const canApply = computed(() => validConditions.value.length > 0)
const canSave = computed(() => canApply.value && name.value.trim().length > 0)

function buildPayload(): FilterQueryPayload {
  return {
    operator: operator.value,
    conditions: validConditions.value.map(c => ({ ...c }))
  }
}

function apply() {
  if (!canApply.value) return
  applying.value = true
  try {
    emit('apply', buildPayload())
    isOpen.value = false
  } finally {
    applying.value = false
  }
}

async function saveAndApply() {
  if (!canSave.value || !auth.account?.id) return
  saving.value = true
  try {
    const body = {
      name: name.value.trim(),
      filter_type: props.filterType,
      query: buildPayload()
    }
    const url = props.editingId
      ? `/accounts/${auth.account.id}/custom_filters/${props.editingId}`
      : `/accounts/${auth.account.id}/custom_filters`
    const method = props.editingId ? 'PATCH' : 'POST'
    const saved = await api<SavedFilter>(url, { method, body })
    savedFilters.upsert(saved)
    emit('save', saved)
    emit('apply', buildPayload())
    toast.add({ title: t('savedFilters.saved'), color: 'success' })
    isOpen.value = false
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    const msg = e?.response?._data?.error ?? t('common.error')
    toast.add({ title: msg, color: 'error' })
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (isOpen.value) hydrateFromInitial()
})
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="t(editingId ? 'savedFilters.edit' : 'savedFilters.advancedFilter')"
    :ui="{ content: 'max-w-3xl' }"
  >
    <template #body>
      <div class="space-y-4">
        <div class="space-y-3">
          <template v-for="(cond, idx) in conditions" :key="idx">
            <div v-if="idx > 0" class="flex items-center">
              <USelect
                v-model="operator"
                :items="connectorItems"
                value-key="value"
                :disabled="idx > 1"
                size="xs"
                class="w-20"
              />
            </div>
            <FiltersConditionRow
              v-model="conditions[idx]!"
              :attributes="attributes"
              :removable="conditions.length > 1"
              @remove="removeCondition(idx)"
            />
          </template>
        </div>

        <UButton
          icon="i-lucide-plus"
          :label="t('savedFilters.addCondition')"
          color="neutral"
          variant="ghost"
          size="sm"
          @click="addCondition"
        />

        <div v-if="showSave" class="pt-3 border-t border-default space-y-2">
          <UFormField :label="t('savedFilters.name')">
            <UInput v-model="name" :placeholder="t('savedFilters.placeholders.name')" class="w-full" />
          </UFormField>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <UButton
          v-if="!showSave"
          icon="i-lucide-bookmark"
          color="neutral"
          variant="ghost"
          size="sm"
          :label="t('savedFilters.saveOption')"
          @click="showSave = true"
        />
        <div v-else />
        <div class="flex items-center gap-2">
          <UButton color="neutral" variant="ghost" @click="isOpen = false">
            {{ t('common.cancel') }}
          </UButton>
          <UButton
            v-if="showSave"
            :loading="saving"
            :disabled="!canSave"
            icon="i-lucide-save"
            @click="saveAndApply"
          >
            {{ editingId ? t('common.save') : t('savedFilters.saveAndApply') }}
          </UButton>
          <UButton
            v-else
            :loading="applying"
            :disabled="!canApply"
            icon="i-lucide-check"
            @click="apply"
          >
            {{ t('savedFilters.apply') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
