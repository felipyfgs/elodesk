<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { notificationPreferencesSchema, type NotificationPreferences } from '~/schemas/settings/notifications'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const errorHandler = useErrorHandler()
const auth = useAuthStore()

const state = reactive<NotificationPreferences>({
  mentions: true,
  assignment: true,
  new_conversation: true,
  unread_message: false,
  sla_breach: true,
  email_enabled: false
})
const loading = ref(false)

const eventSections = computed(() => [
  { key: 'mentions', label: t('settings.notifications.mentions'), icon: 'i-lucide-at-sign' },
  { key: 'assignment', label: t('settings.notifications.assignments'), icon: 'i-lucide-user-check' },
  { key: 'new_conversation', label: t('settings.notifications.newConversation'), icon: 'i-lucide-message-circle-plus' },
  { key: 'unread_message', label: t('settings.notifications.unreadMessage'), icon: 'i-lucide-message-circle-warning' },
  { key: 'sla_breach', label: t('settings.notifications.slaBreach'), icon: 'i-lucide-timer-off' }
] as const)

async function load() {
  if (!auth.user?.id) return
  try {
    const preferences = await api<NotificationPreferences>(`/users/${auth.user.id}/notification_preferences`)
    Object.assign(state, notificationPreferencesSchema.parse({ ...state, ...preferences }))
  } catch (error) {
    if (import.meta.dev) console.warn('[notifications] load failed, using defaults', error)
  }
}

async function onSubmit(event: FormSubmitEvent<NotificationPreferences>) {
  if (!auth.user?.id) return
  loading.value = true
  try {
    await api(`/users/${auth.user.id}/notification_preferences`, {
      method: 'PUT',
      body: event.data
    })
    errorHandler.success(t('common.save'), t('settings.notifications.saved'))
  } catch (error) {
    errorHandler.handle(error, {
      title: t('settings.notifications.saveFailed'),
      onRetry: () => onSubmit(event)
    })
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <UPageCard
    :title="t('settings.notifications.title')"
    :description="t('settings.notifications.description')"
    icon="i-lucide-bell"
    variant="subtle"
  >
    <UForm
      :schema="notificationPreferencesSchema"
      :state="state"
      class="space-y-5"
      @submit="onSubmit"
    >
      <div class="space-y-3">
        <div class="flex items-center justify-between gap-3 rounded-md border border-default px-3 py-2">
          <div class="flex min-w-0 items-center gap-3">
            <UIcon name="i-lucide-monitor-check" class="size-4 shrink-0 text-muted" />
            <span class="text-sm font-medium">{{ t('settings.notifications.inApp') }}</span>
          </div>
          <UBadge :label="t('settings.notifications.alwaysOn')" variant="soft" color="success" />
        </div>

        <UFormField name="email_enabled">
          <div class="flex items-center justify-between gap-3 rounded-md border border-default px-3 py-2">
            <div class="flex min-w-0 items-center gap-3">
              <UIcon name="i-lucide-mail" class="size-4 shrink-0 text-muted" />
              <span class="text-sm font-medium">{{ t('settings.notifications.email') }}</span>
            </div>
            <USwitch v-model="state.email_enabled" />
          </div>
        </UFormField>
      </div>

      <USeparator />

      <div class="space-y-3">
        <UFormField
          v-for="section in eventSections"
          :key="section.key"
          :name="section.key"
        >
          <div class="flex items-center justify-between gap-3 rounded-md border border-default px-3 py-2">
            <div class="flex min-w-0 items-center gap-3">
              <UIcon :name="section.icon" class="size-4 shrink-0 text-muted" />
              <span class="text-sm font-medium">{{ section.label }}</span>
            </div>
            <USwitch v-model="state[section.key]" />
          </div>
        </UFormField>
      </div>

      <div class="flex justify-end">
        <UButton type="submit" icon="i-lucide-save" :loading="loading">
          {{ t('common.save') }}
        </UButton>
      </div>
    </UForm>
  </UPageCard>
</template>
