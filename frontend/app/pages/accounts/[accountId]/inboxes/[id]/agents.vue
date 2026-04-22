<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Agent } from '~/stores/agents'
import type { Inbox, InboxAgent } from '~/stores/inboxes'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const inboxAgents = ref<InboxAgent[]>([])
const accountAgents = ref<Agent[]>([])
const formState = reactive({ userIds: [] as number[] })
const loading = ref(true)
const saving = ref(false)

const activeAccountAgents = computed(() => accountAgents.value.filter(agent => agent.userId > 0))
const selectedAgents = computed(() => activeAccountAgents.value.filter(agent => formState.userIds.includes(Number(agent.userId))))

async function loadAgents() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const [inboxAgentsRes, accountAgentsRes] = await Promise.all([
      api<InboxAgent[]>(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}/agents`),
      api<Agent[]>(`/accounts/${auth.account.id}/agents`)
    ])
    inboxAgents.value = inboxAgentsRes
    accountAgents.value = accountAgentsRes
    formState.userIds = inboxAgentsRes.map(agent => Number(agent.userId))
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
      body: { userIds: formState.userIds }
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
  <UPageCard
    :title="t('inboxes.agents')"
    :description="t('inboxes.agentsDescription')"
    variant="subtle"
  >
    <div v-if="loading" class="flex items-center justify-center py-8">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>

    <template v-else>
      <UForm :state="formState" class="flex flex-col gap-4" @submit="saveAgents">
        <UFormField :label="t('inboxes.agentsSelect')" name="userIds">
          <USelectMenu
            v-model="formState.userIds"
            :items="activeAccountAgents"
            value-key="userId"
            label-key="name"
            multiple
            :search-input="{ placeholder: t('common.search') }"
            class="w-full"
          />
        </UFormField>
      </UForm>

      <div v-if="selectedAgents.length" class="mt-4 grid gap-2 sm:grid-cols-2">
        <div
          v-for="agent in selectedAgents"
          :key="agent.userId"
          class="flex items-center gap-3 rounded-lg bg-elevated px-3 py-2"
        >
          <UAvatar :alt="agent.name" size="xs" />
          <div class="min-w-0">
            <p class="text-sm font-medium truncate">
              {{ agent.name }}
            </p>
            <p class="text-xs text-muted truncate">
              {{ agent.email }}
            </p>
          </div>
        </div>
      </div>

      <UAlert
        v-else
        class="mt-4"
        color="neutral"
        variant="subtle"
        icon="i-lucide-users"
        :title="t('inboxes.noAgents')"
      />

      <div class="flex justify-end mt-4">
        <UButton :loading="saving" @click="saveAgents">
          {{ t('common.save') }}
        </UButton>
      </div>
    </template>
  </UPageCard>
</template>
