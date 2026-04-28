<script setup lang="ts">
import { agentInviteSchema, type AgentInviteForm } from '~/schemas/settings/agents'
import { useAgentsStore } from '~/stores/agents'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const { t } = useI18n()
const toast = useToast()
const store = useAgentsStore()

const state = reactive<Partial<AgentInviteForm>>({ email: '', role: 'agent', name: '' })
const loading = ref(false)

async function onSubmit() {
  loading.value = true
  try {
    await store.invite(state.email ?? '', state.role ?? 'agent', state.name || undefined)
    toast.add({ title: t('settings.agents.inviteSent'), color: 'success' })
    emit('update:open', false)
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal :open="props.open" :title="t('settings.agents.invite')" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-6">
        <UForm
          :schema="agentInviteSchema"
          :state="state"
          class="space-y-4"
          @submit="onSubmit"
        >
          <UFormField :label="t('settings.agents.inviteEmail')" name="email">
            <UInput v-model="state.email" type="email" required />
          </UFormField>
          <UFormField :label="t('settings.general.name')" name="name">
            <UInput v-model="state.name" />
          </UFormField>
          <UFormField :label="t('settings.agents.role')" name="role">
            <SettingsAgentsRoleSelect v-model="state.role as string" />
          </UFormField>
          <div class="flex justify-end gap-2">
            <UButton variant="outline" @click="emit('update:open', false)">
              {{ t('common.cancel') }}
            </UButton>
            <UButton type="submit" :loading="loading">
              {{ t('settings.agents.invite') }}
            </UButton>
          </div>
        </UForm>
      </div>
    </template>
  </UModal>
</template>
