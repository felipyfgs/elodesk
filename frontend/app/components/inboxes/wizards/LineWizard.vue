<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { lineStepSetupSchema, lineStepCredentialsSchema, type LineInboxForm } from '~/schemas/inboxes/line'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<LineInboxForm>({
  name: '',
  lineChannelId: '',
  lineChannelSecret: '',
  lineChannelToken: ''
})

const loading = ref(false)
const setupFormRef = ref()
const credentialsFormRef = ref()

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-message-circle', slot: 'setup' },
  { title: t('inboxes.wizards.credentials'), icon: 'i-lucide-key', slot: 'credentials' }
]

async function validateStep(step: number): Promise<boolean> {
  if (step === 0) {
    const { error } = await lineStepSetupSchema.safeParseAsync({ name: state.name })
    if (error) {
      setupFormRef.value?.setErrors(error.issues.map(i => ({ message: i.message, path: i.path.join('.') })))
      return false
    }
    return true
  }
  if (step === 1) {
    const { error } = await lineStepCredentialsSchema.safeParseAsync({
      lineChannelId: state.lineChannelId,
      lineChannelSecret: state.lineChannelSecret,
      lineChannelToken: state.lineChannelToken
    })
    if (error) {
      credentialsFormRef.value?.setErrors(error.issues.map(i => ({ message: i.message, path: i.path.join('.') })))
      return false
    }
    return true
  }
  return true
}

async function submit() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<{ id: number }>(`/accounts/${auth.account.id}/inboxes/line`, {
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
    :title="t('inboxes.channels.line')"
    :description="t('inboxes.wizards.line.description')"
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
          :schema="lineStepSetupSchema"
          :state="state"
          class="flex flex-col gap-4"
        >
          <UFormField :label="t('inboxes.wizards.name')" name="name" required>
            <UInput v-model="state.name" />
          </UFormField>
        </UForm>
      </UPageCard>
    </template>

    <template #credentials>
      <UPageCard variant="subtle">
        <UForm
          ref="credentialsFormRef"
          :schema="lineStepCredentialsSchema"
          :state="state"
          class="flex flex-col gap-4"
        >
          <UFormField :label="t('inboxes.wizards.line.lineChannelId')" name="lineChannelId" required>
            <UInput v-model="state.lineChannelId" />
          </UFormField>
          <UFormField :label="t('inboxes.wizards.line.lineChannelSecret')" name="lineChannelSecret" required>
            <UTextarea v-model="state.lineChannelSecret" :rows="2" />
          </UFormField>
          <UFormField :label="t('inboxes.wizards.line.lineChannelToken')" name="lineChannelToken" required>
            <UTextarea v-model="state.lineChannelToken" :rows="3" />
          </UFormField>
        </UForm>
      </UPageCard>
    </template>
  </InboxesWizardsBaseWizard>
</template>
