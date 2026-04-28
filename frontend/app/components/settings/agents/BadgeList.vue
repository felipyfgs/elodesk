<script setup lang="ts">
import type { Agent } from '~/stores/agents'

defineProps<{ members: Agent[] }>()
</script>

<template>
  <ul class="flex flex-wrap gap-2">
    <li v-for="m in members" :key="m.id">
      <UBadge :label="m.name" variant="soft" />
    </li>
  </ul>
</template>
