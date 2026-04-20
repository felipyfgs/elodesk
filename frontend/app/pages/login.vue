<script setup lang="ts">
import { z } from 'zod/v4'
import type { FormSubmitEvent, AuthFormField, ButtonProps } from '@nuxt/ui'

const { login, verifyMfa } = useAuth()
const route = useRoute()
const { t } = useI18n()

const providers: ButtonProps[] = []

const fields: AuthFormField[] = [
  { name: 'email', type: 'email', label: t('auth.login.email'), placeholder: t('auth.login.placeholders.email'), required: true },
  { name: 'password', type: 'password', label: t('auth.login.password'), placeholder: t('auth.login.placeholders.password'), required: true },
  { name: 'remember', type: 'checkbox', label: t('auth.login.remember') }
]

const schema = z.object({
  email: z.string().email(t('auth.login.emailInvalid')),
  password: z.string().min(1, t('auth.login.passwordRequired')),
  remember: z.boolean().optional()
})

type Schema = z.output<typeof schema>

const error = ref<string | null>(null)
const loading = ref(false)

// MFA step
const mfaRequired = ref(false)
const mfaToken = ref('')
const mfaCode = ref('')
const mfaLoading = ref(false)
const mfaError = ref<string | null>(null)
const useRecoveryCode = ref(false)
const recoveryCode = ref('')
const mfaAttempts = ref(0)
const mfaLocked = ref(false)
const mfaCountdown = ref(0)

const mfaPinValue = computed<number[]>({
  get: () => mfaCode.value ? mfaCode.value.split('').map(value => Number(value)) : [],
  set: (value) => {
    mfaCode.value = value.map(code => String(code)).join('')
  }
})

let countdownTimer: ReturnType<typeof setInterval> | null = null

function lockMfa() {
  mfaLocked.value = true
  mfaCountdown.value = 300 // 5 minutes
  countdownTimer = setInterval(() => {
    mfaCountdown.value--
    if (mfaCountdown.value <= 0) {
      mfaLocked.value = false
      mfaAttempts.value = 0
      if (countdownTimer) clearInterval(countdownTimer)
    }
  }, 1000)
}

async function onSubmit(event: FormSubmitEvent<Schema>) {
  error.value = null
  loading.value = true
  try {
    const res = await login(event.data.email, event.data.password)
    if ('mfaRequired' in res && res.mfaRequired) {
      mfaRequired.value = true
      mfaToken.value = res.mfaToken
    } else {
      const redirect = (route.query.redirect as string) || '/conversations'
      await navigateTo(redirect)
    }
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    error.value = e?.data?.message === 'invalid credentials'
      ? t('auth.errors.invalidCredentials')
      : (e?.data?.message ?? t('common.error'))
  } finally {
    loading.value = false
  }
}

async function onMfaSubmit() {
  if (mfaLocked.value) return
  const code = useRecoveryCode.value ? recoveryCode.value : mfaCode.value
  if (!code) return

  mfaError.value = null
  mfaLoading.value = true
  try {
    await verifyMfa(mfaToken.value, code)
    const redirect = (route.query.redirect as string) || '/conversations'
    await navigateTo(redirect)
  } catch {
    mfaAttempts.value++
    if (mfaAttempts.value >= 3) {
      lockMfa()
      mfaError.value = t('auth.login.mfaTooManyAttempts')
    } else {
      mfaError.value = t('auth.login.mfaInvalidCode')
    }
  } finally {
    mfaLoading.value = false
  }
}

function onMfaCodeComplete(code: Array<string | number>) {
  mfaCode.value = code.map(value => String(value)).join('')
  onMfaSubmit()
}

function backToLogin() {
  mfaRequired.value = false
  mfaToken.value = ''
  mfaCode.value = ''
  recoveryCode.value = ''
  mfaError.value = null
  mfaAttempts.value = 0
  if (countdownTimer) clearInterval(countdownTimer)
}

onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer)
})

definePageMeta({ layout: 'auth' })
</script>

<template>
  <UPageCard class="w-full max-w-sm">
    <!-- Login step -->
    <UAuthForm
      v-if="!mfaRequired"
      :schema="schema"
      :fields="fields"
      :providers="providers"
      :loading="loading"
      icon="i-lucide-lock"
      :title="t('auth.login.title')"
      :description="t('auth.login.description')"
      :submit="{ label: t('auth.login.submit') }"
      @submit="onSubmit"
    >
      <template #password-hint>
        <NuxtLink to="/forgot-password" class="text-primary font-medium">
          {{ t('auth.forgot.title') }}
        </NuxtLink>
      </template>

      <template #validation>
        <UAlert
          v-if="error"
          icon="i-lucide-circle-alert"
          color="error"
          variant="subtle"
          :title="error"
        />
      </template>

      <template #footer>
        <p class="text-center text-sm text-muted">
          {{ t('auth.login.noAccount') }}
          <NuxtLink to="/register" class="text-primary font-medium">
            {{ t('auth.login.register') }}
          </NuxtLink>
        </p>
      </template>
    </UAuthForm>

    <!-- MFA step -->
    <div v-else class="flex flex-col items-center gap-4">
      <div class="flex flex-col items-center gap-1">
        <div class="flex size-10 items-center justify-center rounded-full bg-primary/10 text-primary mb-2">
          <UIcon name="i-lucide-shield-check" class="size-5" />
        </div>
        <h2 class="text-lg font-semibold text-default">
          {{ t('auth.login.mfaRequired') }}
        </h2>
        <p class="text-sm text-muted">
          {{ t('auth.login.mfaPlaceholder') }}
        </p>
      </div>

      <template v-if="!useRecoveryCode">
        <div class="flex justify-center gap-2">
          <UPinInput
            v-model="mfaPinValue"
            type="number"
            otp
            autofocus
            :length="6"
            size="lg"
            placeholder="○"
            :ui="{ root: 'justify-center gap-2', base: 'font-mono' }"
            @complete="onMfaCodeComplete"
          />
        </div>
      </template>

      <template v-else>
        <UFormField :label="t('auth.login.recoveryCodeLabel')" class="w-full">
          <UInput
            v-model="recoveryCode"
            :placeholder="t('auth.login.recoveryCodePlaceholder')"
          />
        </UFormField>
        <UButton
          :loading="mfaLoading"
          :disabled="mfaLocked"
          block
          @click="onMfaSubmit"
        >
          {{ t('auth.login.verifyMfa') }}
        </UButton>
      </template>

      <UAlert
        v-if="mfaError"
        icon="i-lucide-circle-alert"
        color="error"
        variant="subtle"
        :title="mfaError"
        class="w-full"
      />

      <UAlert
        v-if="mfaLocked"
        icon="i-lucide-clock"
        color="warning"
        variant="subtle"
        class="w-full"
      >
        <template #title>
          {{ t('auth.login.mfaLockedMessage', { min: Math.floor(mfaCountdown / 60), sec: String(mfaCountdown % 60).padStart(2, '0') }) }}
        </template>
      </UAlert>

      <div class="flex flex-col items-center gap-1">
        <UButton
          v-if="!useRecoveryCode"
          variant="ghost"
          size="xs"
          @click="useRecoveryCode = true"
        >
          {{ t('auth.login.useRecoveryCode') }}
        </UButton>
        <UButton
          v-else
          variant="ghost"
          size="xs"
          @click="useRecoveryCode = false"
        >
          {{ t('auth.login.useAuthenticator') }}
        </UButton>
        <UButton variant="ghost" size="xs" @click="backToLogin">
          {{ t('auth.forgot.backToLogin') }}
        </UButton>
      </div>
    </div>
  </UPageCard>
</template>
