<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Inbox } from '~/stores/inboxes'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const agents = ref<Array<{ id: string, userId: string, user?: { id: string, name: string, avatarUrl?: string | null } }>>([])
const accountAgents = ref<Array<{ id: string, name: string, avatarUrl?: string | null }>>([])
const selectedIds = ref<string[]>([])
const loading = ref(true)
const saving = ref(false)

async function loadAgents() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const [inboxAgentsRes, accountAgentsRes] = await Promise.all([
      api<Array<{ id: string, userId: string, user?: { id: string, name: string, avatarUrl?: string | null } }>>(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}/agents`),
      api<Array<{ id: string, name: string, avatarUrl?: string | null }>>(`/accounts/${auth.account.id}/agents`)
    ])
    agents.value = inboxAgentsRes
    accountAgents.value = accountAgentsRes
    selectedIds.value = inboxAgentsRes.map(a => a.userId)
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}

async function saveAgents() {
  if (!auth.account?.id) return
  saving.value = true
  try {
    await api(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}/agents`, {
      method: 'PUT',
      body: { userIds: selectedIds.value.map(Number) }
    })
    toast.add({ title: t('common.success'), color: 'success' })
    await loadAgents()
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    saving.value = false
  }
}

onMounted(loadAgents)
</script>

<template>
  <UPageCard :title="t('inboxes.agents')" variant="subtle">
    <div v-if="loading" class="flex items-center justify-center py-8">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>

    <template v-else>
      <UFormField :label="t('inboxes.agentsSelect')">
        <USelectMenu
          v-model="selectedIds"
          :options="accountAgents"
          value-key="id"
          option-attribute="name"
          multiple
          searchable
          class="w-full"
        />
      </UFormField>

      <div v-if="agents.length" class="mt-4 flex flex-col gap-2">
        <div
          v-for="agent in agents"
          :key="agent.id"
          class="flex items-center gap-3 rounded-lg bg-[var(--ui-bg-accented)] px-3 py-2"
        >
          <UAvatar
            :src="agent.user?.avatarUrl ?? undefined"
            :alt="agent.user?.name"
            size="xs"
          />
          <span class="text-sm font-medium">{{ agent.user?.name }}</span>
          <span class="text-xs text-muted ml-auto">{{ agent.userId }}</span>
        </div>
      </div>

      <div class="flex justify-end mt-4">
        <UButton :loading="saving" @click="saveAgents">
          {{ t('common.save') }}
        </UButton>
      </div>
    </template>
  </UPageCard>
</template>
