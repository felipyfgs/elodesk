<script setup lang="ts">
import type { EntityMetric } from '~/types/reports'

defineProps<{ items: EntityMetric[] }>()
const { t } = useI18n()
</script>

<template>
  <UCard :ui="{ header: 'flex items-center justify-between' }">
    <template #header>
      <h3 class="font-semibold">
        {{ t('reports.topAgents') }}
      </h3>
    </template>

    <UEmpty
      v-if="!items.length"
      icon="i-lucide-users"
      :title="t('reports.noAgentsData')"
      variant="subtle"
      size="sm"
    />

    <ul v-else class="divide-y divide-default">
      <li v-for="a in items.slice(0, 5)" :key="a.entityId" class="flex items-center justify-between py-2">
        <UUser :name="a.entityName" size="sm" />
        <UBadge :label="String(a.total)" variant="soft" size="sm" />
      </li>
    </ul>
  </UCard>
</template>
