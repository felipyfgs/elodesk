<script setup lang="ts">
defineProps<{ metric: { total: number, open: number, resolved: number } | null }>()
</script>

<template>
  <div v-if="metric" class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <UPageCard title="Total" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ metric.total }}
      </p>
    </UPageCard>
    <UPageCard title="Open" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ metric.open }}
      </p>
    </UPageCard>
    <UPageCard title="Resolved" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ metric.resolved }}
      </p>
    </UPageCard>
  </div>
</template>
