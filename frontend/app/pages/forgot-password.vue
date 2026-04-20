<script setup lang="ts">
import { z } from 'zod/v4'
import type { FormSubmitEvent, AuthFormField } from '@nuxt/ui'

const { t } = useI18n()
const api = useApi()

const fields: AuthFormField[] = [
  {
    name: 'email',
    type: 'email',
    label: t('auth.forgot.email'),
    placeholder: t('auth.forgot.placeholders.email'),
    required: true
  }
]

const schema = z.object({
  email: z.email(t('auth.login.emailInvalid'))
})

type Schema = z.output<typeof schema>
const state = reactive<Partial<Schema>>({ email: '' })
const loading = ref(false)
const sent = ref(false)

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    await api('/auth/forgot', { method: 'POST', body: { email: event.data.email } })
    sent.value = true
  } catch {
    // Always show success message per spec (never leak email existence).
    sent.value = true
  } finally {
    loading.value = false
  }
}

definePageMeta({ layout: 'auth' })
</script>

<template>
  <UPageCard
    v-if="!sent"
    class="w-full max-w-sm"
  >
    <UAuthForm
      :schema="schema"
      :state="state"
      :fields="fields"
      :loading="loading"
      icon="i-lucide-key-round"
      :title="t('auth.forgot.title')"
      :description="t('auth.forgot.description')"
      :submit="{ label: t('auth.forgot.submit') }"
      @submit="onSubmit"
    >
      <template #footer>
        <p class="text-center text-sm text-muted">
          {{ t('auth.forgot.backToLogin') }}
          <NuxtLink to="/login" class="text-primary font-medium">
            {{ t('auth.login.submit') }}
          </NuxtLink>
        </p>
      </template>
    </UAuthForm>
  </UPageCard>

  <UPageCard v-else class="w-full max-w-sm" :title="t('auth.forgot.title')">
    <div class="text-center py-4">
      <UIcon name="i-lucide-check-circle" class="text-4xl text-primary mb-2" />
      <p class="text-sm text-muted">
        {{ t('auth.forgot.success') }}
      </p>
    </div>

    <template #footer>
      <p class="text-center text-sm text-muted">
        <NuxtLink to="/login" class="text-primary font-medium">
          {{ t('auth.forgot.backToLogin') }}
        </NuxtLink>
      </p>
    </template>
  </UPageCard>
</template>
