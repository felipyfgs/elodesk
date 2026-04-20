<script setup lang="ts">
definePageMeta({ layout: 'dashboard' })

// Redirect /reports → /reports/overview
await navigateTo('/reports/overview', { replace: true })
</script>

<template>
  <div />
</template>
