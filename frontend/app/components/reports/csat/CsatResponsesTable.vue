<script setup lang="ts">
defineProps<{ items?: Array<{ id: number, score: number, comment: string, createdAt: string }> }>()
</script>

<template>
  <UPageCard title="Recent" variant="subtle">
    <p class="text-sm text-muted text-center py-6">
      —
    </p>
  </UPageCard>
</template>
