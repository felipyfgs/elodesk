<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: false })

const auth = useAuthStore()
if (import.meta.client) auth.hydrate()

const target = (() => {
  if (!auth.isAuthenticated) return '/login'
  const primaryId = auth.accounts[0]?.id
  return primaryId ? `/accounts/${primaryId}/conversations` : '/login'
})()

await navigateTo(target, { replace: true })
</script>

<template>
  <div />
</template>
