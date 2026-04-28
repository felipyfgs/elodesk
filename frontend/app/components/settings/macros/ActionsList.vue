<script setup lang="ts">
import type { MacroAction } from '~/schemas/settings/macros'

const props = defineProps<{ modelValue: MacroAction[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: MacroAction[]] }>()

const { t } = useI18n()

function addAction() {
  emit('update:modelValue', [...props.modelValue, { name: 'assign_agent', params: {} }])
}

function updateAt(index: number, value: MacroAction) {
  const next = [...props.modelValue]
  next[index] = value
  emit('update:modelValue', next)
}

function removeAt(index: number) {
  const next = [...props.modelValue]
  next.splice(index, 1)
  emit('update:modelValue', next)
}
</script>

<template>
  <div class="space-y-3">
    <SettingsMacrosActionItem
      v-for="(action, i) in props.modelValue"
      :key="i"
      :model-value="action"
      @update:model-value="updateAt(i, $event)"
      @remove="removeAt(i)"
    />
    <UButton variant="outline" icon="i-lucide-plus" @click="addAction">
      {{ t('settings.macros.actions') }}
    </UButton>
  </div>
</template>
