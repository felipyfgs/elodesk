<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import {
  twilioStepMediumSchema,
  twilioStepCredentialsSchema,
  twilioStepSenderSchema,
  type TwilioInboxForm
} from '~/schemas/inboxes/twilio'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()
const runtimeConfig = useRuntimeConfig()

const state = reactive<TwilioInboxForm>({
  name: '',
  medium: 'whatsapp',
  accountSid: '',
  authToken: '',
  apiKeySid: '',
  phoneNumber: '',
  messagingServiceSid: ''
})

const loading = ref(false)
const step = ref(0)

const smsMediumEnabled = computed<boolean>(() => {
  const raw = runtimeConfig.public?.featureTwilioSmsMedium
  return raw === true || raw === 'true'
})

const mediumOptions = computed(() => {
  const opts = [{ value: 'whatsapp', label: 'WhatsApp' }]
  if (smsMediumEnabled.value) {
    opts.push({ value: 'sms', label: 'SMS' })
  }
  return opts
})

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-message-square', slot: 'setup' },
  { title: t('inboxes.wizards.credentials'), icon: 'i-lucide-key', slot: 'credentials' },
  { title: t('inboxes.wizards.twilio.sender'), icon: 'i-lucide-phone', slot: 'sender' }
]

const setupFormRef = ref()
const credentialsFormRef = ref()
const senderFormRef = ref()

async function validateCurrentStep(): Promise<boolean> {
  if (step.value === 0) {
    const { error } = await twilioStepMediumSchema.safeParseAsync({
      name: state.name, medium: state.medium
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
    const { error } = await twilioStepCredentialsSchema.safeParseAsync({
      accountSid: state.accountSid,
      authToken: state.authToken,
      apiKeySid: state.apiKeySid
    })
    if (error) {
      credentialsFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }
  if (step.value === 2) {
    const { error } = await twilioStepSenderSchema.safeParseAsync({
      phoneNumber: state.phoneNumber,
      messagingServiceSid: state.messagingServiceSid
    })
    if (error) {
      senderFormRef.value?.setErrors(
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

async function submit() {
  if (!auth.account?.id) return
  if (!(await validateCurrentStep())) return
  loading.value = true
  try {
    const res = await api<{ id: number }>(
      `/accounts/${auth.account.id}/inboxes/twilio`,
      {
        method: 'POST',
        body: {
          name: state.name,
          medium: state.medium,
          accountSid: state.accountSid,
          authToken: state.authToken,
          apiKeySid: state.apiKeySid || undefined,
          phoneNumber: state.phoneNumber || undefined,
          messagingServiceSid: state.messagingServiceSid || undefined
        }
      }
    )
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
        {{ t('inboxes.channels.twilio') }}
      </h2>
      <p class="text-sm text-muted">
        {{ t('inboxes.wizards.twilio.description') }}
      </p>
    </div>

    <UStepper v-model="step" :items="stepperItems" :linear="true">
      <template #setup="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="setupFormRef"
            :schema="twilioStepMediumSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.name')" name="name" required>
              <UInput v-model="state.name" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.twilio.medium')" name="medium" required>
              <USelect
                v-model="state.medium"
                :items="mediumOptions"
                value-key="value"
                label-key="label"
              />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #credentials="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="credentialsFormRef"
            :schema="twilioStepCredentialsSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.twilio.accountSid')" name="accountSid" required>
              <UInput v-model="state.accountSid" />
            </UFormField>
            <UFormField :label="t('inboxes.wizards.twilio.authToken')" name="authToken" required>
              <UTextarea v-model="state.authToken" :rows="2" />
            </UFormField>
            <UFormField :label="t('inboxes.wizards.twilio.apiKeySid')" name="apiKeySid">
              <UInput v-model="state.apiKeySid" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #sender="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="senderFormRef"
            :schema="twilioStepSenderSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UAlert
              icon="i-lucide-info"
              color="info"
              variant="subtle"
              :description="t('inboxes.wizards.twilio.senderXorHint')"
            />
            <UFormField :label="t('inboxes.wizards.twilio.phoneNumber')" name="phoneNumber">
              <UInput v-model="state.phoneNumber" placeholder="+14155551234" />
            </UFormField>
            <UFormField :label="t('inboxes.wizards.twilio.messagingServiceSid')" name="messagingServiceSid">
              <UInput v-model="state.messagingServiceSid" placeholder="MGxxxxxxxxxxxx" />
            </UFormField>
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
