<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <UApp>
    <div class="min-h-dvh flex flex-col items-center justify-center p-4 bg-elevated/50">
      <div class="mb-8 flex flex-col items-center gap-2">
        <UIcon name="i-lucide-message-square" class="size-8 text-primary" />
        <span class="text-lg font-semibold text-default">{{ t('app.name') }}</span>
      </div>
      <slot />
    </div>
  </UApp>
</template>
