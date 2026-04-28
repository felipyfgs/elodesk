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
  <footer class="flex h-14 shrink-0 items-center justify-between gap-3 border-t border-default bg-elevated px-3 sm:px-4">
    <div class="flex items-center gap-3">
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="md"
        :aria-label="t('conversations.forward.cancel')"
        @click="emit('cancel')"
      />
      <span class="text-sm font-medium text-highlighted">
        {{ t('conversations.forward.selectionTitle', { count }) }}
      </span>
    </div>
    <UButton
      icon="i-lucide-corner-up-right"
      color="primary"
      variant="ghost"
      size="md"
      :disabled="count === 0"
      :aria-label="t('conversations.forward.triggerAction')"
      @click="emit('forward')"
    />
  </footer>
</template>
