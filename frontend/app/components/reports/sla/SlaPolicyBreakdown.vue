<script setup lang="ts">
defineProps<{ items: Array<{ policyId: number, policyName: string, total: number, breached: number }> }>()
</script>

<template>
  <UPageCard title="Por política" variant="subtle">
    <ul class="divide-y divide-default">
      <li v-for="p in items" :key="p.policyId" class="py-2 flex items-center justify-between">
        <span class="text-sm">{{ p.policyName }}</span>
        <span class="text-sm text-muted">{{ p.total - p.breached }}/{{ p.total }}</span>
      </li>
    </ul>
  </UPageCard>
</template>
