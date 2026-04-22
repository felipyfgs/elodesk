<script setup lang="ts">
import ProfileAvatarField from '~/components/settings/profile/ProfileAvatarField.vue'
import ProfilePasswordCard from '~/components/settings/profile/ProfilePasswordCard.vue'
import ProfileAccessTokenCard from '~/components/settings/profile/ProfileAccessTokenCard.vue'
import ProfileDangerZone from '~/components/settings/profile/ProfileDangerZone.vue'
import ProfileNotificationPreferencesCard from '~/components/settings/profile/ProfileNotificationPreferencesCard.vue'
import { profileSchema, type ProfileForm } from '~/schemas/settings/profile'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()

const state = reactive<Partial<ProfileForm>>({
  name: auth.user?.name ?? '',
  email: auth.user?.email ?? '',
  avatarUrl: undefined
})
const loading = ref(false)

async function onSubmit() {
  if (!auth.user?.id) return
  loading.value = true
  try {
    await api(`/users/${auth.user.id}`, {
      method: 'PUT',
      body: { name: state.name, email: state.email, avatarUrl: state.avatarUrl }
    })
    if (state.name) auth.user.name = state.name
    if (state.email) auth.user.email = state.email
    toast.add({ title: t('settings.profile.title'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <UPageCard :title="t('settings.profile.title')" variant="subtle">
      <UForm
        :schema="profileSchema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <ProfileAvatarField v-model="state.avatarUrl" />
        <UFormField :label="t('settings.general.name')" name="name">
          <UInput v-model="state.name" />
        </UFormField>
        <UFormField :label="t('settings.general.email')" name="email">
          <UInput v-model="state.email" type="email" />
        </UFormField>
        <div class="flex justify-end">
          <UButton type="submit" :loading="loading">
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UPageCard>

    <ProfilePasswordCard />
    <ProfileNotificationPreferencesCard />
    <ProfileAccessTokenCard />
    <ProfileDangerZone />
  </div>
</template>
