<script setup lang="ts">
import type { EntityMetric } from '~/types/reports'
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const aid = computed(() => auth.account?.id ?? '')

defineProps<{ items: EntityMetric[], entity: string }>()
</script>

<template>
  <div class="border border-default rounded-lg overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-muted text-left">
        <tr>
          <th class="px-4 py-2 font-medium">
            Name
          </th>
          <th class="px-4 py-2 font-medium">
            Total
          </th>
          <th class="px-4 py-2 font-medium">
            Open
          </th>
          <th class="px-4 py-2 font-medium">
            Resolved
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="m in items" :key="m.entityId" class="border-t border-default">
          <td class="px-4 py-2">
            <NuxtLink :to="aid ? `/accounts/${aid}/reports/${entity}/${m.entityId}` : `/reports/${entity}/${m.entityId}`" class="hover:underline">
              {{ m.entityName }}
            </NuxtLink>
          </td>
          <td class="px-4 py-2 text-muted">
            {{ m.total }}
          </td>
          <td class="px-4 py-2 text-muted">
            {{ m.open }}
          </td>
          <td class="px-4 py-2 text-muted">
            {{ m.resolved }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
