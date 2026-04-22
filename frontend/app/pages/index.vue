<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const auth = useAuthStore()
const primaryId = auth.accounts[0]?.id

if (primaryId) {
  navigateTo(`/accounts/${primaryId}`, { replace: true })
} else if (!auth.isAuthenticated) {
  navigateTo('/login', { replace: true })
}
</script>

<template>
  <div />
</template>
