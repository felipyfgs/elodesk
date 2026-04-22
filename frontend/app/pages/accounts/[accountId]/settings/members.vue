<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <UPageCard :title="t('settings.members.title')" variant="subtle">
    <p class="text-sm text-muted">
      {{ t('settings.members.empty') }}
    </p>
  </UPageCard>
</template>
