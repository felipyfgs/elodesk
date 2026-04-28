<script setup lang="ts">
const route = useRoute()
const aid = route.params.accountId as string

await navigateTo(`/accounts/${aid}/conversations`, { replace: true })
</script>

<template>
  <div />
</template>
