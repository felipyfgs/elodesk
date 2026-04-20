<script setup lang="ts">
defineProps<{ data?: Array<{ score: number, count: number }> }>()
</script>

<template>
  <UPageCard title="Distribution" variant="subtle">
    <div class="h-40 flex items-center justify-center text-muted text-sm">
      CSAT disabled — enable in settings to collect scores
    </div>
  </UPageCard>
</template>
