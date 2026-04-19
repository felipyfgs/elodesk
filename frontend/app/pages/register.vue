<script setup lang="ts">
const { register } = useAuth()
const { t } = useI18n()

const email = ref('')
const password = ref('')
const name = ref('')
const accountName = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit() {
  error.value = null
  loading.value = true
  try {
    await register({
      email: email.value,
      password: password.value,
      name: name.value,
      accountName: accountName.value || undefined
    })
    await navigateTo('/sessions')
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    if (e?.data?.message?.includes('email already')) {
      error.value = t('auth.errors.emailExists')
    } else {
      error.value = e?.data?.message ?? 'erro'
    }
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
          {{ t('auth.register.title') }}
        </h1>
      </template>
      <form class="flex flex-col gap-3" @submit.prevent="onSubmit">
        <UFormField :label="t('auth.register.name')">
          <UInput v-model="name" required />
        </UFormField>
        <UFormField :label="t('auth.register.email')">
          <UInput
            v-model="email"
            type="email"
            required
            autocomplete="email"
          />
        </UFormField>
        <UFormField :label="t('auth.register.password')">
          <UInput
            v-model="password"
            type="password"
            minlength="8"
            required
          />
        </UFormField>
        <UFormField :label="t('auth.register.accountName')">
          <UInput v-model="accountName" />
        </UFormField>
        <p v-if="error" class="text-sm text-error">
          {{ error }}
        </p>
        <UButton type="submit" :loading="loading" block>
          {{ t('auth.register.submit') }}
        </UButton>
      </form>
      <template #footer>
        <p class="text-sm text-muted">
          {{ t('auth.register.haveAccount') }}
          <ULink to="/login">
            {{ t('auth.register.login') }}
          </ULink>
        </p>
      </template>
    </UCard>
  </div>
</template>
