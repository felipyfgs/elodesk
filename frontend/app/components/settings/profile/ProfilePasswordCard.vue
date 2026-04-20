<script setup lang="ts">
import { passwordChangeSchema, type PasswordChangeForm } from '~/schemas/settings/profile'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()

const state = reactive<Partial<PasswordChangeForm>>({ currentPassword: '', newPassword: '', confirmPassword: '' })
const loading = ref(false)

async function onSubmit() {
  if (!auth.user?.id) return
  loading.value = true
  try {
    await api(`/users/${auth.user.id}`, {
      method: 'PUT',
      body: { currentPassword: state.currentPassword, newPassword: state.newPassword }
    })
    toast.add({ title: t('settings.profile.changePassword'), color: 'success' })
    state.currentPassword = ''
    state.newPassword = ''
    state.confirmPassword = ''
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UPageCard :title="t('settings.profile.changePassword')" variant="subtle">
    <UForm
      :schema="passwordChangeSchema"
      :state="state"
      class="space-y-4"
      @submit="onSubmit"
    >
      <UFormField :label="t('settings.profile.currentPassword')" name="currentPassword">
        <UInput v-model="state.currentPassword" type="password" autocomplete="current-password" />
      </UFormField>
      <UFormField :label="t('settings.profile.newPassword')" name="newPassword">
        <UInput v-model="state.newPassword" type="password" autocomplete="new-password" />
      </UFormField>
      <UFormField :label="t('settings.profile.confirmPassword')" name="confirmPassword">
        <UInput v-model="state.confirmPassword" type="password" autocomplete="new-password" />
      </UFormField>
      <div class="flex justify-end">
        <UButton type="submit" :loading="loading">
          {{ t('common.save') }}
        </UButton>
      </div>
    </UForm>
  </UPageCard>
</template>
