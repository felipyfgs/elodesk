<script setup lang="ts">
import { ConfirmModal } from '#components'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const confirm = useOverlay().create(ConfirmModal)
const auth = useAuthStore()

const token = ref<string | null>(null)
const loading = ref(false)
const resetting = ref(false)

async function fetchToken() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<{ token: string }>(`/accounts/${auth.account.id}/profile/access_token`)
    token.value = res.token
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}

async function copyToken() {
  if (!token.value) return
  await navigator.clipboard.writeText(token.value)
  toast.add({ title: t('settings.profile.tokenCopied'), color: 'success' })
}

async function confirmReset() {
  const confirmed = await confirm.open({
    title: t('settings.profile.resetTokenConfirm'),
    description: t('settings.profile.resetTokenConfirmDesc'),
    confirmLabel: t('settings.profile.resetToken'),
    confirmColor: 'error'
  })
  if (!confirmed) return

  if (!auth.account?.id) return
  resetting.value = true
  try {
    const res = await api<{ token: string }>(`/accounts/${auth.account.id}/profile/access_token/reset`, { method: 'POST' })
    token.value = res.token
    toast.add({ title: t('settings.profile.resetTokenSuccess'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    resetting.value = false
  }
}

onMounted(fetchToken)
</script>

<template>
  <UPageCard :title="t('settings.profile.accessToken')" :description="t('settings.profile.accessTokenDesc')" variant="subtle">
    <div v-if="loading" class="flex items-center gap-2">
      <UIcon name="i-lucide-loader-circle" class="animate-spin text-muted" />
    </div>
    <div v-else class="flex items-center gap-2">
      <UInput
        :model-value="token ?? ''"
        type="password"
        readonly
        class="font-mono flex-1"
      />
      <UButton
        variant="outline"
        :icon="'i-lucide-copy'"
        :aria-label="t('settings.profile.copyToken')"
        @click="copyToken"
      >
        {{ t('settings.profile.copyToken') }}
      </UButton>
      <UButton
        variant="outline"
        color="error"
        :icon="'i-lucide-rotate-ccw'"
        :loading="resetting"
        :aria-label="t('settings.profile.resetToken')"
        @click="confirmReset"
      >
        {{ t('settings.profile.resetToken') }}
      </UButton>
    </div>
  </UPageCard>
</template>
