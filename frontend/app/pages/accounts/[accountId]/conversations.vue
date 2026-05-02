<script setup lang="ts">
definePageMeta({
  layout: 'dashboard',
  middleware: 'conversations-scope',
  key: () => '/conversations'
})
</script>

<template>
  <Conversations />
  <NuxtPage />
</template>
