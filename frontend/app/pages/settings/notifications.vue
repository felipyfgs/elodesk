<script setup lang="ts">
import { notificationPreferencesSchema, type NotificationPreferences } from '~/schemas/settings/notifications'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()

const prefs = reactive<NotificationPreferences>({
  mentions: true, assignment: true, new_conversation: true,
  unread_message: false, sla_breach: true, email_enabled: false
})
const loading = ref(false)

const eventSections = computed(() => [
  { key: 'mentions', label: t('settings.notifications.mentions') },
  { key: 'assignment', label: t('settings.notifications.assignments') },
  { key: 'new_conversation', label: t('settings.notifications.newConversation') },
  { key: 'unread_message', label: 'Unread messages' },
  { key: 'sla_breach', label: t('settings.notifications.slaBreach') }
])

async function load() {
  if (!auth.user?.id) return
  try {
    const res = await api<NotificationPreferences>(`/users/${auth.user.id}/notification_preferences`)
    if (res && typeof res === 'object') {
      Object.assign(prefs, notificationPreferencesSchema.parse({ ...prefs, ...res }))
    }
  } catch {
    // leave defaults
  }
}

async function save() {
  if (!auth.user?.id) return
  loading.value = true
  try {
    await api(`/users/${auth.user.id}/notification_preferences`, {
      method: 'PUT', body: prefs
    })
    toast.add({ title: t('common.save'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <UPageCard :title="t('settings.notifications.title')" variant="subtle">
      <template #description>
        Canais
      </template>
      <div class="flex items-center justify-between py-2">
        <span>in-app</span>
        <UBadge label="always on" variant="soft" />
      </div>
      <div class="flex items-center justify-between py-2">
        <span>{{ t('settings.notifications.email') }}</span>
        <USwitch v-model="prefs.email_enabled" />
      </div>
    </UPageCard>

    <UPageCard title="Eventos" variant="subtle">
      <div v-for="section in eventSections" :key="section.key" class="flex items-center justify-between py-2 border-b border-default last:border-b-0">
        <span>{{ section.label }}</span>
        <USwitch v-model="prefs[section.key as keyof NotificationPreferences]" />
      </div>
      <template #footer>
        <div class="flex justify-end">
          <UButton :loading="loading" @click="save">
            {{ t('common.save') }}
          </UButton>
        </div>
      </template>
    </UPageCard>
  </div>
</template>
