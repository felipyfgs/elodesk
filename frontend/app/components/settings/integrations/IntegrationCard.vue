<script setup lang="ts">
defineProps<{ title: string, description: string, icon?: string, status?: string }>()
</script>

<template>
  <UPageCard :title="title" :description="description" variant="subtle">
    <template #leading>
      <UIcon v-if="icon" :name="icon" class="text-xl" />
    </template>
    <UBadge
      v-if="status"
      :label="status"
      variant="soft"
      size="sm"
    />
  </UPageCard>
</template>
