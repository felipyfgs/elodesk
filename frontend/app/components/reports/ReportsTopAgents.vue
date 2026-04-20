<script setup lang="ts">
import type { EntityMetric } from '~/types/reports'

defineProps<{ items: EntityMetric[] }>()
const { t: _t } = useI18n()
</script>

<template>
  <UPageCard title="Top agents" variant="subtle">
    <ul class="divide-y divide-default">
      <li v-for="a in items.slice(0, 5)" :key="a.entityId" class="flex items-center justify-between py-2">
        <span class="text-sm">{{ a.entityName }}</span>
        <UBadge :label="String(a.total)" variant="soft" size="sm" />
      </li>
    </ul>
  </UPageCard>
</template>
