<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const primaryId = auth.accounts[0]?.id

if (primaryId) {
  await navigateTo(`/accounts/${primaryId}/conversations`, { replace: true })
} else {
  await navigateTo('/login', { replace: true })
}
</script>

<template>
  <div />
</template>
