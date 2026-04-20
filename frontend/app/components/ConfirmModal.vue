<script setup lang="ts">
const props = withDefaults(defineProps<{
  title?: string
  description?: string
  itemName?: string
  confirmLabel?: string
  cancelLabel?: string
  confirmColor?: 'error' | 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'neutral'
  loading?: boolean
}>(), {
  title: undefined,
  description: undefined,
  itemName: undefined,
  confirmLabel: undefined,
  cancelLabel: undefined,
  confirmColor: 'error',
  loading: false
})

const emit = defineEmits<{
  close: [confirmed: boolean]
}>()

const { t } = useI18n()
</script>

<template>
  <UModal
    :title="props.title ?? t('common.confirm')"
    :description="props.description"
    @update:open="emit('close', false)"
  >
    <template #body>
      <p v-if="props.itemName" class="text-sm font-medium text-default">
        {{ props.itemName }}
      </p>
      <slot />
    </template>

    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="props.loading"
          @click="emit('close', false)"
        >
          {{ props.cancelLabel ?? t('common.cancel') }}
        </UButton>
        <UButton
          :color="props.confirmColor"
          :loading="props.loading"
          @click="emit('close', true)"
        >
          {{ props.confirmLabel ?? t('common.confirm') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
