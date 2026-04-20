<script setup lang="ts">
import type { Period } from '~/types/reports'

const props = defineProps<{ modelValue: Period }>()
const emit = defineEmits<{ 'update:modelValue': [value: Period] }>()

const options = [
  { label: 'Daily', value: 'daily' },
  { label: 'Weekly', value: 'weekly' },
  { label: 'Monthly', value: 'monthly' }
]

const model = computed({
  get: () => props.modelValue,
  set: (v: Period) => emit('update:modelValue', v)
})
</script>

<template>
  <USelect
    v-model="model"
    :items="options"
    value-key="value"
    size="sm"
  />
</template>
