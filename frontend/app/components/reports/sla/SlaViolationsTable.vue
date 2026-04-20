<script setup lang="ts">
defineProps<{ items: Array<{ policyId: number, policyName: string, breached: number, total: number }> }>()
</script>

<template>
  <UPageCard title="Top violações" variant="subtle">
    <div class="border border-default rounded-md overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted text-left">
          <tr>
            <th class="px-4 py-2 font-medium">
              Política
            </th>
            <th class="px-4 py-2 font-medium">
              Breached
            </th>
            <th class="px-4 py-2 font-medium">
              Total
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in items.slice(0, 10)" :key="p.policyId" class="border-t border-default">
            <td class="px-4 py-2">
              {{ p.policyName }}
            </td>
            <td class="px-4 py-2 text-error font-semibold">
              {{ p.breached }}
            </td>
            <td class="px-4 py-2 text-muted">
              {{ p.total }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </UPageCard>
</template>
