<script setup lang="ts">
import type { Range } from '~/types/reports'

const props = defineProps<{ modelValue: Range }>()
const emit = defineEmits<{ 'update:modelValue': [value: Range] }>()
const { t } = useI18n()

function fmt(d: Date): string {
  return d.toISOString().slice(0, 10)
}

const start = computed({
  get: () => fmt(props.modelValue.start),
  set: (v: string) => emit('update:modelValue', { ...props.modelValue, start: new Date(v) })
})
const end = computed({
  get: () => fmt(props.modelValue.end),
  set: (v: string) => emit('update:modelValue', { ...props.modelValue, end: new Date(v) })
})
</script>

<template>
  <div class="flex items-center gap-2">
    <UInput
      v-model="start"
      type="date"
      size="sm"
      :placeholder="t('reports.from')"
    />
    <span class="text-muted">→</span>
    <UInput
      v-model="end"
      type="date"
      size="sm"
      :placeholder="t('reports.to')"
    />
  </div>
</template>
