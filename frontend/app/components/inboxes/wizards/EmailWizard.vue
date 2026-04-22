<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { emailStepSetupSchema, emailStepImapSchema, emailStepSmtpSchema, type EmailInboxForm } from '~/schemas/inboxes/email'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<EmailInboxForm>({
  name: '',
  provider: 'generic',
  email: '',
  imapAddress: '',
  imapPort: 993,
  imapLogin: '',
  imapPassword: '',
  imapEnableSsl: true,
  imapEnabled: true,
  smtpAddress: '',
  smtpPort: 587,
  smtpLogin: '',
  smtpPassword: '',
  smtpEnableSsl: true
})

const loading = ref(false)
const step = ref(0)
const isOAuth = computed(() => state.provider === 'google' || state.provider === 'microsoft')

const stepperItems = computed<StepperItem[]>(() => {
  const items: StepperItem[] = [
    { title: t('inboxes.wizards.email.setup'), icon: 'i-lucide-mail', slot: 'setup' }
  ]
  if (!isOAuth.value) {
    items.push(
      { title: 'IMAP', icon: 'i-lucide-server', slot: 'imap' },
      { title: 'SMTP', icon: 'i-lucide-send', slot: 'smtp' }
    )
  }
  return items
})

const setupFormRef = ref()
const imapFormRef = ref()
const smtpFormRef = ref()

async function validateCurrentStep(): Promise<boolean> {
  if (isOAuth.value) {
    const { error } = await emailStepSetupSchema.safeParseAsync({
      name: state.name,
      provider: state.provider,
      email: state.email
    })
    if (error) {
      setupFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }

  if (step.value === 0) {
    const { error } = await emailStepSetupSchema.safeParseAsync({
      name: state.name,
      provider: state.provider,
      email: state.email
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
    const { error } = await emailStepImapSchema.safeParseAsync({
      imapAddress: state.imapAddress,
      imapPort: state.imapPort,
      imapLogin: state.imapLogin,
      imapPassword: state.imapPassword,
      imapEnableSsl: state.imapEnableSsl
    })
    if (error) {
      imapFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }

  if (step.value === 2) {
    const { error } = await emailStepSmtpSchema.safeParseAsync({
      smtpAddress: state.smtpAddress,
      smtpPort: state.smtpPort,
      smtpLogin: state.smtpLogin,
      smtpPassword: state.smtpPassword,
      smtpEnableSsl: state.smtpEnableSsl
    })
    if (error) {
      smtpFormRef.value?.setErrors(
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
    const res = await api<{ inboxId: number, authorizeUrl?: string }>(`/accounts/${auth.account.id}/inboxes/email`, {
      method: 'POST',
      body: state
    })
    if (res.authorizeUrl) {
      toast.add({ title: t('inboxes.wizards.email.redirecting'), color: 'info' })
      window.location.href = res.authorizeUrl
    } else {
      toast.add({ title: t('common.success'), color: 'success' })
      router.push(`/accounts/${auth.account?.id}/inboxes/${res.inboxId}`)
    }
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}

const isLastStep = computed(() => step.value >= stepperItems.value.length - 1)
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h2 class="text-lg font-semibold">
        {{ t('inboxes.channels.email') }}
      </h2>
      <p class="text-sm text-muted">
        {{ t('inboxes.wizards.email.description') }}
      </p>
    </div>

    <UStepper v-model="step" :items="stepperItems" :linear="true">
      <template #setup="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="setupFormRef"
            :schema="emailStepSetupSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.name')" name="name" required>
              <UInput v-model="state.name" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.provider')" name="provider" required>
              <USelect
                v-model="state.provider"
                :items="[
                  { value: 'generic', label: 'IMAP/SMTP' },
                  { value: 'google', label: 'Google' },
                  { value: 'microsoft', label: 'Microsoft' }
                ]"
                value-key="value"
                label-key="label"
              />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.email')" name="email" required>
              <UInput v-model="state.email" type="email" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #imap="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="imapFormRef"
            :schema="emailStepImapSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.email.imapAddress')" name="imapAddress" required>
              <UInput v-model="state.imapAddress" placeholder="imap.example.com" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.imapPort')" name="imapPort" required>
              <UInput v-model.number="state.imapPort" type="number" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.imapLogin')" name="imapLogin">
              <UInput v-model="state.imapLogin" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.imapPassword')" name="imapPassword" required>
              <UInput v-model="state.imapPassword" type="password" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.imapEnableSsl')" name="imapEnableSsl">
              <UCheckbox v-model="state.imapEnableSsl" :label="t('inboxes.wizards.email.enableSsl')" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #smtp="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="smtpFormRef"
            :schema="emailStepSmtpSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.email.smtpAddress')" name="smtpAddress" required>
              <UInput v-model="state.smtpAddress" placeholder="smtp.example.com" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.smtpPort')" name="smtpPort" required>
              <UInput v-model.number="state.smtpPort" type="number" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.smtpLogin')" name="smtpLogin">
              <UInput v-model="state.smtpLogin" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.smtpPassword')" name="smtpPassword" required>
              <UInput v-model="state.smtpPassword" type="password" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.email.smtpEnableSsl')" name="smtpEnableSsl">
              <UCheckbox v-model="state.smtpEnableSsl" :label="t('inboxes.wizards.email.enableSsl')" />
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
        {{ isOAuth ? t('inboxes.wizards.email.authorize') : t('common.create') }}
      </UButton>
    </div>
  </div>
</template>
