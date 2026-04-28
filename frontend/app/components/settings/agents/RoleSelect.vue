<script setup lang="ts">
const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const { t: _t } = useI18n()

const options = [
  { label: 'Agent', value: 'agent' },
  { label: 'Admin', value: 'admin' },
  { label: 'Owner', value: 'owner' }
]

const model = computed({
  get: () => props.modelValue,
  set: (v: string) => emit('update:modelValue', v)
})
</script>

<template>
  <USelect v-model="model" :items="options" value-key="value" />
</template>
