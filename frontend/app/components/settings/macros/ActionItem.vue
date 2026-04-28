<script setup lang="ts">
import type { MacroAction } from '~/schemas/settings/macros'

const props = defineProps<{ modelValue: MacroAction }>()
const emit = defineEmits<{ 'update:modelValue': [value: MacroAction], 'remove': [] }>()

const { t: _t } = useI18n()

const actionOptions = [
  { label: 'Assign Agent', value: 'assign_agent' },
  { label: 'Assign Team', value: 'assign_team' },
  { label: 'Add Label', value: 'add_label' },
  { label: 'Remove Label', value: 'remove_label' },
  { label: 'Change Status', value: 'change_status' },
  { label: 'Snooze Until', value: 'snooze_until' },
  { label: 'Send Message', value: 'send_message' },
  { label: 'Add Note', value: 'add_note' }
]

const name = computed({
  get: () => props.modelValue.name,
  set: v => emit('update:modelValue', { ...props.modelValue, name: v })
})
const params = computed({
  get: () => JSON.stringify(props.modelValue.params ?? {}, null, 2),
  set: (v) => {
    try {
      const parsed = v ? JSON.parse(v) : {}
      emit('update:modelValue', { ...props.modelValue, params: parsed })
    } catch {
      // keep previous
    }
  }
})
</script>

<template>
  <div class="flex gap-2 items-start border border-default rounded-md p-3">
    <div class="flex-1 space-y-2">
      <USelect v-model="name" :items="actionOptions" value-key="value" />
      <UTextarea
        v-model="params"
        :rows="3"
        class="font-mono text-xs"
        placeholder="{}"
      />
    </div>
    <UButton
      icon="i-lucide-trash"
      variant="ghost"
      color="error"
      size="xs"
      @click="emit('remove')"
    />
  </div>
</template>
