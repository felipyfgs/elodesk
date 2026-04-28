<script setup lang="ts">
// Parent route for /conversations and all sub-scopes (team, label, inbox,
// filter, unattended, :conversationId). Mounting the Conversations page
// component here keeps the list panel, scroll position and realtime
// subscription alive across child navigations — children are routing-only
// placeholders. Auto-import name is <Conversations> (Nuxt drops the "Index"
// suffix for files named Index.vue inside a folder).
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
