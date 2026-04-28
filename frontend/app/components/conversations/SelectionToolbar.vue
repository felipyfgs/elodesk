<script setup lang="ts">
defineProps<{
  count: number
}>()

const emit = defineEmits<{
  cancel: []
  forward: []
}>()

const { t } = useI18n()
</script>

<template>
  <header class="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-default bg-elevated px-3 sm:px-4">
    <div class="flex items-center gap-2">
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="sm"
        :aria-label="t('conversations.forward.cancel')"
        @click="emit('cancel')"
      />
      <span class="text-sm font-medium text-highlighted">
        {{ t('conversations.forward.selectionTitle', { count }) }}
      </span>
    </div>
    <div class="flex items-center gap-2">
      <UButton
        color="neutral"
        variant="ghost"
        size="sm"
        @click="emit('cancel')"
      >
        {{ t('conversations.forward.cancel') }}
      </UButton>
      <UButton
        :disabled="count === 0"
        color="primary"
        size="sm"
        @click="emit('forward')"
      >
        {{ t('conversations.forward.submitButton', { count }) }}
      </UButton>
    </div>
  </header>
</template>
