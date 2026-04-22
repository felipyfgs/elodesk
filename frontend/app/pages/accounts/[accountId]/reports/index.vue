<script setup lang="ts">
definePageMeta({ layout: 'dashboard' })

const route = useRoute()
const accountId = route.params.accountId as string
await navigateTo(`/accounts/${accountId}/reports/overview`, { replace: true })
</script>

<template>
  <div />
</template>
