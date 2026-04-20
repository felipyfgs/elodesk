<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <div class="flex items-center justify-center gap-2 text-sm text-muted">
    <slot />
    <ULink to="/login" class="text-primary font-medium">
      {{ t('auth.forgot.backToLogin') }}
    </ULink>
  </div>
</template>
