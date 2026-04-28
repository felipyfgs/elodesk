<script setup lang="ts">
defineProps<{ score?: number | null }>()
</script>

<template>
  <UPageCard title="CSAT" variant="subtle">
    <p class="text-3xl font-semibold">
      {{ score != null ? score.toFixed(1) : '—' }}
    </p>
  </UPageCard>
</template>
