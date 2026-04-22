<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { smsStepSetupSchema, smsStepProviderSchema, type SmsInboxForm } from '~/schemas/inboxes/sms'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<SmsInboxForm>({
  name: '',
  provider: 'bandwidth',
  phoneNumber: '',
  bandwidth: { accountId: '', applicationId: '', basicAuthUser: '', basicAuthPass: '' },
  zenvia: { apiToken: '' }
})

const loading = ref(false)
const step = ref(0)

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-message-square', slot: 'setup' },
  { title: t('inboxes.wizards.providerConfig'), icon: 'i-lucide-key', slot: 'providerConfig' }
]

const setupFormRef = ref()
const providerFormRef = ref()

async function validateCurrentStep(): Promise<boolean> {
  if (step.value === 0) {
    const { error } = await smsStepSetupSchema.safeParseAsync({
      name: state.name,
      provider: state.provider,
      phoneNumber: state.phoneNumber
    })
    if (error) {
      setupFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }

  if (step.value === 1) {
    const { error } = await smsStepProviderSchema.safeParseAsync({
      bandwidth: state.bandwidth,
      zenvia: state.zenvia
    })
    if (error) {
      providerFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }

  return true
}

async function nextStep() {
  if (!(await validateCurrentStep())) return
  step.value++
}

function getPayload() {
  const { provider, name, phoneNumber } = state
  const providerConfig: Record<string, Record<string, string>> = {}
  if (provider === 'bandwidth' && state.bandwidth) providerConfig.bandwidth = state.bandwidth
  else if (provider === 'zenvia' && state.zenvia) providerConfig.zenvia = state.zenvia
  return { name, provider, phoneNumber, providerConfig }
}

async function submit() {
  if (!auth.account?.id) return
  if (!(await validateCurrentStep())) return
  loading.value = true
  try {
    const res = await api<{ id: number }>(`/accounts/${auth.account.id}/inboxes/sms`, {
      method: 'POST',
      body: getPayload()
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

const isLastStep = computed(() => step.value >= stepperItems.length - 1)
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h2 class="text-lg font-semibold">
        {{ t('inboxes.channels.sms') }}
      </h2>
      <p class="text-sm text-muted">
        {{ t('inboxes.wizards.sms.description') }}
      </p>
    </div>

    <UStepper v-model="step" :items="stepperItems" :linear="true">
      <template #setup="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="setupFormRef"
            :schema="smsStepSetupSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.name')" name="name" required>
              <UInput v-model="state.name" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.sms.provider')" name="provider" required>
              <USelect
                v-model="state.provider"
                :items="[
                  { value: 'bandwidth', label: 'Bandwidth' },
                  { value: 'zenvia', label: 'Zenvia' }
                ]"
                value-key="value"
                label-key="label"
              />
            </UFormField>

            <UAlert
              icon="i-lucide-info"
              color="info"
              variant="subtle"
              :description="t('inboxes.wizards.sms.twilioMovedDescription')"
            />

            <UFormField :label="t('inboxes.wizards.sms.phoneNumber')" name="phoneNumber" required>
              <UInput v-model="state.phoneNumber" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #providerConfig="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="providerFormRef"
            :schema="smsStepProviderSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <template v-if="state.provider === 'bandwidth'">
              <UFormField :label="t('inboxes.wizards.sms.bandwidth.accountId')" name="bandwidth.accountId" required>
                <UInput v-model="state.bandwidth!.accountId" />
              </UFormField>
              <UFormField :label="t('inboxes.wizards.sms.bandwidth.applicationId')" name="bandwidth.applicationId" required>
                <UInput v-model="state.bandwidth!.applicationId" />
              </UFormField>
              <UFormField :label="t('inboxes.wizards.sms.bandwidth.username')" name="bandwidth.basicAuthUser" required>
                <UInput v-model="state.bandwidth!.basicAuthUser" />
              </UFormField>
              <UFormField :label="t('inboxes.wizards.sms.bandwidth.password')" name="bandwidth.basicAuthPass" required>
                <UInput v-model="state.bandwidth!.basicAuthPass" type="password" />
              </UFormField>
            </template>

            <template v-if="state.provider === 'zenvia'">
              <UFormField :label="t('inboxes.wizards.sms.zenvia.apiToken')" name="zenvia.apiToken" required>
                <UTextarea v-model="state.zenvia!.apiToken" :rows="2" />
              </UFormField>
            </template>
          </UForm>
        </UPageCard>
      </template>
    </UStepper>

    <div class="flex justify-end gap-2">
      <UButton :to="`/accounts/${auth.account?.id}/inboxes/new`" variant="ghost" color="neutral">
        {{ t('common.cancel') }}
      </UButton>
      <UButton v-if="!isLastStep" :disabled="loading" @click="nextStep">
        {{ t('common.next') }}
      </UButton>
      <UButton
        v-else
        type="button"
        :loading="loading"
        @click="submit"
      >
        {{ t('common.create') }}
      </UButton>
    </div>
  </div>
</template>
