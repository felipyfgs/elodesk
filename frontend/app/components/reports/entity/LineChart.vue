<script setup lang="ts">
import { VisXYContainer, VisLine, VisAxis } from '@unovis/vue'

defineProps<{ data: Array<{ x: number, y: number }> }>()
const xAcc = (d: { x: number }) => d.x
const yAcc = (d: { y: number }) => d.y
</script>

<template>
  <UPageCard title="Timeline" variant="subtle">
    <div class="h-64">
      <VisXYContainer :data="data" class="h-full w-full">
        <VisLine :x="xAcc" :y="yAcc" />
        <VisAxis type="x" />
        <VisAxis type="y" />
      </VisXYContainer>
    </div>
  </UPageCard>
</template>
