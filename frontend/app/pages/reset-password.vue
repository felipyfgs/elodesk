<script setup lang="ts">
import { resetSchema } from '~/schemas/auth/reset'
import type { ResetForm } from '~/schemas/auth/reset'
import type { FormSubmitEvent, AuthFormField } from '@nuxt/ui'

const { t } = useI18n()
const api = useApi()
const route = useRoute()
const toast = useToast()

const token = route.query.token as string

const state = reactive<ResetForm>({ password: '', confirm: '' })
const fields: AuthFormField[] = [
  {
    name: 'password',
    type: 'password',
    label: t('auth.reset.newPassword'),
    placeholder: t('auth.reset.placeholders.password'),
    required: true
  },
  {
    name: 'confirm',
    type: 'password',
    label: t('auth.reset.confirmPassword'),
    placeholder: t('auth.reset.placeholders.confirmPassword'),
    required: true
  }
]

const loading = ref(false)
const tokenValid = ref<boolean | null>(null)
const validating = ref(true)

onMounted(async () => {
  if (!token) {
    tokenValid.value = false
    validating.value = false
    return
  }

  try {
    await api<{ valid: boolean }>(`/auth/reset/${token}/validate`)
    tokenValid.value = true
  } catch {
    tokenValid.value = false
  } finally {
    validating.value = false
  }
})

async function onSubmit(event: FormSubmitEvent<ResetForm>) {
  loading.value = true
  try {
    await api('/auth/reset', { method: 'POST', body: { token, newPassword: event.data.password } })
    toast.add({ title: t('auth.reset.success'), color: 'success' })
    await navigateTo('/login')
  } catch {
    toast.add({ title: t('auth.reset.invalidToken'), color: 'error' })
  } finally {
    loading.value = false
  }
}

definePageMeta({ layout: 'auth' })
</script>

<template>
  <div v-if="validating" class="flex items-center justify-center">
    <UIcon name="i-lucide-loader-2" class="animate-spin text-2xl text-muted" />
  </div>

  <UPageCard
    v-else-if="tokenValid"
    class="w-full max-w-sm"
  >
    <UAuthForm
      :schema="resetSchema"
      :state="state"
      :fields="fields"
      :loading="loading"
      icon="i-lucide-lock-keyhole"
      :title="t('auth.reset.title')"
      :description="t('auth.reset.description')"
      :submit="{ label: t('auth.reset.submit') }"
      @submit="onSubmit"
    >
      <template #footer>
        <p class="text-center text-sm text-muted">
          <NuxtLink to="/login" class="text-primary font-medium">
            {{ t('auth.reset.backToLogin') }}
          </NuxtLink>
        </p>
      </template>
    </UAuthForm>
  </UPageCard>

  <UPageCard v-else class="w-full max-w-sm" :title="t('auth.reset.title')">
    <div class="text-center py-4">
      <UIcon name="i-lucide-triangle-alert" class="text-4xl text-warning mb-2" />
      <p class="text-sm text-muted">
        {{ t('auth.reset.invalidToken') }}
      </p>
      <UButton
        variant="outline"
        class="mt-4"
        to="/forgot-password"
      >
        {{ t('auth.reset.backToLogin') }}
      </UButton>
    </div>
  </UPageCard>
</template>
