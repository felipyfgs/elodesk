<script setup lang="ts">
const props = withDefaults(defineProps<{
  title?: string
  description?: string
  itemName?: string
  confirmLabel?: string
  cancelLabel?: string
  confirmColor?: 'error' | 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'neutral'
  loading?: boolean
  confirmValue?: string
  confirmPlaceholder?: string
}>(), {
  title: undefined,
  description: undefined,
  itemName: undefined,
  confirmLabel: undefined,
  cancelLabel: undefined,
  confirmColor: 'error',
  loading: false,
  confirmValue: undefined,
  confirmPlaceholder: undefined
})

const emit = defineEmits<{
  close: [confirmed: boolean]
}>()

const { t } = useI18n()
const inputValue = ref('')

const canConfirm = computed(() => {
  if (!props.confirmValue) return true
  return inputValue.value.trim() === props.confirmValue
})

function handleClose() {
  inputValue.value = ''
  emit('close', false)
}

function handleConfirm() {
  if (!canConfirm.value) return
  inputValue.value = ''
  emit('close', true)
}
</script>

<template>
  <UModal
    :title="props.title ?? t('common.confirm')"
    :description="props.description"
    @update:open="handleClose"
  >
    <template #body>
      <div class="space-y-3">
        <p v-if="props.itemName" class="text-sm font-medium text-default">
          {{ props.itemName }}
        </p>
        <slot />
        <UFormField v-if="props.confirmValue" :label="props.confirmPlaceholder ?? t('common.typeToConfirm')">
          <UInput
            v-model="inputValue"
            :placeholder="props.confirmValue"
            class="w-full"
          />
        </UFormField>
        <p
          v-if="props.confirmValue && inputValue && !canConfirm"
          class="text-xs text-error"
        >
          {{ t('common.nameMismatch') }}
        </p>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="props.loading"
          @click="handleClose"
        >
          {{ props.cancelLabel ?? t('common.cancel') }}
        </UButton>
        <UButton
          :color="props.confirmColor"
          :loading="props.loading"
          :disabled="!canConfirm"
          @click="handleConfirm"
        >
          {{ props.confirmLabel ?? t('common.confirm') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
