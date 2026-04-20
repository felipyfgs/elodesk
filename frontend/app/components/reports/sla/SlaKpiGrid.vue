<script setup lang="ts">
defineProps<{ report: { total: number, met: number, breached: number } | null }>()
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <UPageCard title="Total" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ report?.total ?? 0 }}
      </p>
    </UPageCard>
    <UPageCard title="Cumpridas" variant="subtle">
      <p class="text-3xl font-semibold">
        {{ report?.met ?? 0 }}
      </p>
    </UPageCard>
    <UPageCard title="Breached" variant="subtle">
      <p class="text-3xl font-semibold text-error">
        {{ report?.breached ?? 0 }}
      </p>
    </UPageCard>
  </div>
</template>
