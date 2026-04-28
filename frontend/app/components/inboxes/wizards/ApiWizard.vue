<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { apiInboxSchema, type ApiInboxForm } from '~/schemas/inboxes/api'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<ApiInboxForm>({ name: '' })
const loading = ref(false)
const setupFormRef = ref()

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-code', slot: 'setup' }
]

async function validateStep(_step: number): Promise<boolean> {
  const { error } = await apiInboxSchema.safeParseAsync({ name: state.name })
  if (error) {
    setupFormRef.value?.setErrors(
      error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
    )
    return false
  }
  return true
}

async function submit() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<{ id: number }>(`/accounts/${auth.account.id}/inboxes`, {
      method: 'POST',
      body: state
    })
    toast.add({ title: t('common.success'), color: 'success' })
    router.push(`/accounts/${auth.account?.id}/inboxes/${res.id}`)
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <InboxesWizardsBaseWizard
    :title="t('inboxes.channels.api')"
    :description="t('inboxes.wizards.api.description')"
    :steps="stepperItems"
    :cancel-to="`/accounts/${auth.account?.id}/inboxes/new`"
    :validate-step="validateStep"
    :submit="submit"
    :loading="loading"
  >
    <template #setup>
      <UPageCard variant="subtle">
        <UForm
          ref="setupFormRef"
          :schema="apiInboxSchema"
          :state="state"
          class="flex flex-col gap-4"
        >
          <UFormField :label="t('inboxes.wizards.name')" name="name" required>
            <UInput v-model="state.name" />
          </UFormField>
        </UForm>
      </UPageCard>
    </template>
  </InboxesWizardsBaseWizard>
</template>
