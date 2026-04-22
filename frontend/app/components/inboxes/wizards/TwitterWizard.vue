<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const loading = ref(false)

async function connect() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<{ url: string }>(
      `/accounts/${auth.account.id}/inboxes/twitter/authorize`,
      { method: 'POST' }
    )
    if (import.meta.client) {
      window.location.href = res.url
    }
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
    loading.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h2 class="text-lg font-semibold">
        {{ t('inboxes.channels.twitter') }}
      </h2>
      <p class="text-sm text-muted">
        {{ t('inboxes.wizards.twitter.description') }}
      </p>
    </div>

    <UPageCard variant="subtle">
      <UAlert
        icon="i-lucide-dollar-sign"
        color="warning"
        variant="subtle"
        :title="t('inboxes.wizards.twitter.costWarning')"
        :description="t('inboxes.wizards.twitter.costWarningDescription')"
      />
    </UPageCard>

    <div class="flex justify-end gap-2">
      <UButton :to="`/accounts/${auth.account?.id}/inboxes/new`" variant="ghost" color="neutral">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        type="button"
        icon="i-simple-icons-x"
        :loading="loading"
        @click="connect"
      >
        {{ t('inboxes.wizards.twitter.connect') }}
      </UButton>
    </div>
  </div>
</template>
