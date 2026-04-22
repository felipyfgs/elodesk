<script setup lang="ts">
import { tiktokInboxSchema, type TiktokInboxForm } from '~/schemas/inboxes/tiktok'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const state = reactive<TiktokInboxForm>({ name: '' })
const loading = ref(false)
const formRef = ref()

async function authorize() {
  if (!auth.account?.id) return
  const { error } = await tiktokInboxSchema.safeParseAsync(state)
  if (error) {
    formRef.value?.setErrors(
      error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
    )
    return
  }
  loading.value = true
  try {
    const res = await api<{ url: string }>(
      `/accounts/${auth.account.id}/inboxes/tiktok/authorize`,
      { method: 'POST', body: { name: state.name } }
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
        {{ t('inboxes.channels.tiktok') }}
      </h2>
      <p class="text-sm text-muted">
        {{ t('inboxes.wizards.tiktok.description') }}
      </p>
    </div>

    <UPageCard variant="subtle">
      <UForm
        ref="formRef"
        :schema="tiktokInboxSchema"
        :state="state"
        class="flex flex-col gap-4"
      >
        <UFormField :label="t('inboxes.wizards.name')" name="name" required>
          <UInput v-model="state.name" />
        </UFormField>

        <UAlert
          icon="i-lucide-info"
          color="info"
          variant="subtle"
          :title="t('inboxes.wizards.tiktok.requiresApproval')"
          :description="t('inboxes.wizards.tiktok.requiresApprovalDescription')"
        />
      </UForm>
    </UPageCard>

    <div class="flex justify-end gap-2">
      <UButton :to="`/accounts/${auth.account?.id}/inboxes/new`" variant="ghost" color="neutral">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        type="button"
        icon="i-simple-icons-tiktok"
        :loading="loading"
        @click="authorize"
      >
        {{ t('inboxes.wizards.tiktok.connect') }}
      </UButton>
    </div>
  </div>
</template>
