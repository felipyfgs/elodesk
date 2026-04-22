<script setup lang="ts">
const route = useRoute()

await navigateTo(`/accounts/${route.params.accountId}/inboxes/${route.params.id}`, { replace: true })
</script>

<template>
  <div />
</template>
