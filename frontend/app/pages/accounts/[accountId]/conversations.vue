<script setup lang="ts">
// Parent route for /conversations and all sub-scopes (team, label, inbox,
// filter, unattended, :conversationId). Mounting ConversationsIndex here keeps
// the list panel, scroll position and realtime subscription alive across child
// navigations — children are routing-only placeholders.
definePageMeta({
  layout: 'dashboard',
  middleware: 'conversations-scope',
  key: () => '/conversations'
})
</script>

<template>
  <ConversationsIndex />
  <NuxtPage />
</template>
