<script setup lang="ts">
import type { FilterConditionForm } from '~/schemas/savedFilter'
import type { FilterAttribute, FilterOperator } from '~/composables/useFilterAttributes'
import { OPERATORS_NO_INPUT, OPERATORS_MULTI_INPUT, OPERATORS_BY_TYPE } from '~/composables/useFilterAttributes'

const props = defineProps<{
  modelValue: FilterConditionForm
  attributes: FilterAttribute[]
  removable?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: FilterConditionForm]
  'remove': []
}>()

const { operatorsFor } = useFilterAttributes()

const condition = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v)
})

const currentAttribute = computed<FilterAttribute | undefined>(() =>
  props.attributes.find(a => a.key === condition.value.attribute_key)
)

const attributeItems = computed(() =>
  props.attributes.map(a => ({ value: a.key, label: a.label }))
)

const operatorItems = computed(() => {
  const type = currentAttribute.value?.type
  if (!type) return []
  return operatorsFor(type)
})

function onAttributeChange(key: string) {
  const attr = props.attributes.find(a => a.key === key)
  const firstOp = attr ? OPERATORS_BY_TYPE[attr.type][0] : 'equal_to'
  condition.value = {
    attribute_key: key,
    filter_operator: firstOp as FilterOperator,
    value: null
  }
}

function onOperatorChange(op: FilterOperator) {
  let nextValue: unknown = condition.value.value
  const wasNoInput = OPERATORS_NO_INPUT.has(condition.value.filter_operator as FilterOperator)
  const nowNoInput = OPERATORS_NO_INPUT.has(op)
  const wasMulti = OPERATORS_MULTI_INPUT.has(condition.value.filter_operator as FilterOperator)
  const nowMulti = OPERATORS_MULTI_INPUT.has(op)

  if (nowNoInput) nextValue = null
  else if (wasNoInput) nextValue = null
  if (nowMulti !== wasMulti) nextValue = null

  condition.value = {
    ...condition.value,
    filter_operator: op,
    value: nextValue
  }
}

const hideValueInput = computed(() =>
  OPERATORS_NO_INPUT.has(condition.value.filter_operator as FilterOperator)
)

const isMultiValue = computed(() =>
  OPERATORS_MULTI_INPUT.has(condition.value.filter_operator as FilterOperator)
)

const valueType = computed(() => currentAttribute.value?.type)
const valueOptions = computed(() => currentAttribute.value?.options ?? [])

// Normalize the value depending on operator shape (single vs array)
const singleValue = computed<string | number | undefined>({
  get: () => {
    const v = condition.value.value
    if (Array.isArray(v)) return (v[0] ?? undefined) as string | number | undefined
    return (v ?? undefined) as string | number | undefined
  },
  set: (v) => { condition.value = { ...condition.value, value: v ?? null } }
})

const multiValue = computed<(string | number)[]>({
  get: () => {
    const v = condition.value.value
    if (Array.isArray(v)) return v as (string | number)[]
    if (v === null || v === undefined || v === '') return []
    return [v as string | number]
  },
  set: (v) => { condition.value = { ...condition.value, value: v } }
})

const betweenFrom = computed<string>({
  get: () => {
    const v = condition.value.value
    return Array.isArray(v) ? String(v[0] ?? '') : ''
  },
  set: (v) => {
    const arr = Array.isArray(condition.value.value) ? [...condition.value.value] : ['', '']
    arr[0] = v
    condition.value = { ...condition.value, value: arr }
  }
})

const betweenTo = computed<string>({
  get: () => {
    const v = condition.value.value
    return Array.isArray(v) ? String(v[1] ?? '') : ''
  },
  set: (v) => {
    const arr = Array.isArray(condition.value.value) ? [...condition.value.value] : ['', '']
    arr[1] = v
    condition.value = { ...condition.value, value: arr }
  }
})

const dateInputType = computed(() => valueType.value === 'date' ? 'date' : 'text')
const isBetween = computed(() => condition.value.filter_operator === 'between')
</script>

<template>
  <div class="flex items-start gap-2">
    <USelectMenu
      :model-value="condition.attribute_key"
      :items="attributeItems"
      value-key="value"
      :placeholder="$t('savedFilters.placeholders.attribute')"
      class="w-44 shrink-0"
      size="sm"
      @update:model-value="onAttributeChange"
    />

    <USelect
      :model-value="condition.filter_operator"
      :items="operatorItems"
      value-key="value"
      :disabled="!currentAttribute"
      class="w-36 shrink-0"
      size="sm"
      @update:model-value="onOperatorChange"
    />

    <div class="flex-1 min-w-0">
      <!-- No input needed -->
      <UInput
        v-if="hideValueInput"
        :model-value="$t('savedFilters.noValueNeeded')"
        disabled
        class="w-full"
        size="sm"
      />

      <!-- Between: two inputs -->
      <div v-else-if="isBetween" class="flex items-center gap-1.5">
        <UInput
          v-model="betweenFrom"
          :type="dateInputType"
          :placeholder="$t('savedFilters.placeholders.from')"
          class="flex-1"
          size="sm"
        />
        <span class="text-xs text-muted">—</span>
        <UInput
          v-model="betweenTo"
          :type="dateInputType"
          :placeholder="$t('savedFilters.placeholders.to')"
          class="flex-1"
          size="sm"
        />
      </div>

      <!-- Enum with options, multi -->
      <USelectMenu
        v-else-if="valueOptions.length && isMultiValue"
        v-model="multiValue"
        :items="valueOptions"
        value-key="value"
        multiple
        :placeholder="$t('savedFilters.placeholders.value')"
        class="w-full"
        size="sm"
      />

      <!-- Enum with options, single -->
      <USelectMenu
        v-else-if="valueOptions.length"
        v-model="singleValue"
        :items="valueOptions"
        value-key="value"
        :placeholder="$t('savedFilters.placeholders.value')"
        class="w-full"
        size="sm"
      />

      <!-- Free-form multi value (text IN) -->
      <UInput
        v-else-if="isMultiValue"
        :model-value="multiValue.join(',')"
        :placeholder="$t('savedFilters.placeholders.multiValue')"
        class="w-full"
        size="sm"
        @update:model-value="(v: string) => { multiValue = v.split(',').map((s: string) => s.trim()).filter(Boolean) }"
      />

      <!-- Number -->
      <UInput
        v-else-if="valueType === 'number'"
        v-model.number="singleValue"
        type="number"
        :placeholder="$t('savedFilters.placeholders.value')"
        class="w-full"
        size="sm"
      />

      <!-- Date -->
      <UInput
        v-else-if="valueType === 'date'"
        v-model="singleValue"
        type="date"
        class="w-full"
        size="sm"
      />

      <!-- Fallback text -->
      <UInput
        v-else
        v-model="singleValue"
        :placeholder="$t('savedFilters.placeholders.value')"
        class="w-full"
        size="sm"
      />
    </div>

    <UButton
      icon="i-lucide-x"
      size="sm"
      color="neutral"
      variant="ghost"
      :disabled="!removable"
      :aria-label="$t('savedFilters.remove')"
      @click="emit('remove')"
    />
  </div>
</template>
