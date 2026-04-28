<script setup lang="ts">
const props = defineProps<{ inboxId?: string, labelId?: string }>()
const emit = defineEmits<{
  'update:inboxId': [value: string]
  'update:labelId': [value: string]
}>()

const inboxModel = computed({ get: () => props.inboxId ?? '', set: v => emit('update:inboxId', v) })
const labelModel = computed({ get: () => props.labelId ?? '', set: v => emit('update:labelId', v) })
</script>

<template>
  <div class="space-y-2">
    <UFormField label="Inbox ID">
      <UInput v-model="inboxModel" size="sm" />
    </UFormField>
    <UFormField label="Label ID">
      <UInput v-model="labelModel" size="sm" />
    </UFormField>
  </div>
</template>
