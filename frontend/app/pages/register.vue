<script setup lang="ts">
import { z } from 'zod/v4'
import type { FormSubmitEvent, AuthFormField } from '@nuxt/ui'

const { register } = useAuth()
const { t } = useI18n()
const { markSystemSetup } = await import('~/middleware/auth.global')

const fields: AuthFormField[] = [
  { name: 'name', type: 'text', label: t('auth.register.name'), placeholder: t('auth.register.placeholders.name'), required: true },
  { name: 'email', type: 'email', label: t('auth.register.email'), placeholder: t('auth.register.placeholders.email'), required: true },
  { name: 'password', type: 'password', label: t('auth.register.password'), placeholder: t('auth.register.placeholders.password'), required: true },
  { name: 'confirmPassword', type: 'password', label: t('auth.register.confirmPassword'), placeholder: t('auth.register.placeholders.confirmPassword'), required: true },
  { name: 'accountName', type: 'text', label: t('auth.register.accountName'), placeholder: t('auth.register.placeholders.accountName') }
]

const schema = z.object({
  name: z.string().min(1, t('auth.register.nameRequired')),
  email: z.string().email(t('auth.login.emailInvalid')),
  password: z.string().min(8, t('auth.register.passwordMin')),
  confirmPassword: z.string().min(8, t('auth.register.passwordMin')),
  accountName: z.string().optional()
}).refine(data => data.password === data.confirmPassword, {
  path: ['confirmPassword'],
  message: t('auth.register.passwordMismatch')
})

type Schema = z.output<typeof schema>

const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit(event: FormSubmitEvent<Schema>) {
  error.value = null
  loading.value = true
  try {
    await register({
      email: event.data.email,
      password: event.data.password,
      name: event.data.name,
      accountName: event.data.accountName || undefined
    })
    markSystemSetup()
    await navigateTo('/conversations')
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    if (e?.data?.message?.includes('email already')) {
      error.value = t('auth.errors.emailExists')
    } else {
      error.value = e?.data?.message ?? t('common.error')
    }
  } finally {
    loading.value = false
  }
}

definePageMeta({ layout: 'auth' })
</script>

<template>
  <UPageCard class="w-full max-w-md">
    <UAuthForm
      :schema="schema"
      :fields="fields"
      :loading="loading"
      icon="i-lucide-user-plus"
      :title="t('auth.register.title')"
      :description="t('auth.register.description')"
      :submit="{ label: t('auth.register.submit') }"
      @submit="onSubmit"
    >
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
          {{ t('auth.register.haveAccount') }}
          <NuxtLink to="/login" class="text-primary font-medium">
            {{ t('auth.register.login') }}
          </NuxtLink>
        </p>
      </template>
    </UAuthForm>
  </UPageCard>
</template>
