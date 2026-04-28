<script setup lang="ts">
defineProps<{
  showPagination?: boolean
}>()
</script>

<template>
  <div class="flex flex-col h-full bg-default">
    <!-- Header -->
    <slot name="header" />

    <!-- Bulk Toolbar (condicional) -->
    <slot name="bulk-toolbar" />

    <!-- Body -->
    <main class="flex-1 overflow-y-auto px-6 py-4">
      <div class="w-full mx-auto max-w-5xl">
        <slot />
      </div>
    </main>

    <!-- Footer (condicional) -->
    <footer v-if="showPagination" class="sticky bottom-0 z-0 border-t border-default bg-default">
      <slot name="footer" />
    </footer>
  </div>
</template>
