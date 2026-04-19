<script setup lang="ts">
const { login } = useAuth()
const route = useRoute()
const { t } = useI18n()

const email = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit() {
  error.value = null
  loading.value = true
  try {
    await login(email.value, password.value)
    const redirect = (route.query.redirect as string) || '/sessions'
    await navigateTo(redirect)
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    error.value = e?.data?.message === 'invalid credentials'
      ? t('auth.errors.invalidCredentials')
      : (e?.data?.message ?? 'erro')
  } finally {
    loading.value = false
  }
}

definePageMeta({ layout: false })
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-default p-4">
    <UCard class="w-full max-w-sm">
      <template #header>
        <h1 class="text-lg font-semibold">
          {{ t('auth.login.title') }}
        </h1>
      </template>
      <form class="flex flex-col gap-3" @submit.prevent="onSubmit">
        <UFormField :label="t('auth.login.email')">
          <UInput
            v-model="email"
            type="email"
            required
            autocomplete="email"
          />
        </UFormField>
        <UFormField :label="t('auth.login.password')">
          <UInput
            v-model="password"
            type="password"
            required
            autocomplete="current-password"
          />
        </UFormField>
        <p v-if="error" class="text-sm text-error">
          {{ error }}
        </p>
        <UButton type="submit" :loading="loading" block>
          {{ t('auth.login.submit') }}
        </UButton>
      </form>
      <template #footer>
        <p class="text-sm text-muted">
          {{ t('auth.login.noAccount') }}
          <ULink to="/register">
            {{ t('auth.login.register') }}
          </ULink>
        </p>
      </template>
    </UCard>
  </div>
</template>
