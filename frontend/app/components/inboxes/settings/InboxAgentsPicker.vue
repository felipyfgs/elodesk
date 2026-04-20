<script setup lang="ts">
const props = defineProps<{
  agents: Array<{ id: string, name: string, avatarUrl?: string | null }>
  modelValue: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const { t } = useI18n()
</script>

<template>
  <UFormField :label="t('inboxes.agentsSelect')">
    <USelectMenu
      :model-value="props.modelValue"
      :options="props.agents"
      value-key="id"
      option-attribute="name"
      multiple
      searchable
      class="w-full"
      @update:model-value="emit('update:modelValue', $event as string[])"
    />
  </UFormField>
</template>
