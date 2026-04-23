<script setup lang="ts">
import type { Period } from '~/types'

const props = defineProps<{ modelValue: Period }>()
const emit = defineEmits<{ 'update:modelValue': [value: Period] }>()
const { t } = useI18n()

const options = computed(() => [
  { label: t('reports.daily'), value: 'daily' },
  { label: t('reports.weekly'), value: 'weekly' },
  { label: t('reports.monthly'), value: 'monthly' }
])

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
