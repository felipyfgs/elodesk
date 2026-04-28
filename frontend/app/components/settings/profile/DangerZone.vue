<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()

async function logoutAllSessions() {
  auth.clear()
  await router.push('/login')
}
</script>

<template>
  <UPageCard variant="subtle" :title="t('settings.profile.title')">
    <template #description>
      {{ t('auth.logout') }}
    </template>
    <UButton color="error" variant="outline" @click="logoutAllSessions">
      {{ t('auth.logout') }}
    </UButton>
  </UPageCard>
</template>
