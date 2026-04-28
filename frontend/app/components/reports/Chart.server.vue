<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <div class="space-y-6">
    <UCard :ui="{ header: 'flex items-center justify-between' }">
      <template #header>
        <h3 class="font-semibold">
          {{ t('reports.volume') }}
        </h3>
      </template>
      <div class="h-72 flex items-center justify-center text-muted text-sm">
        …
      </div>
    </UCard>
  </div>
</template>
