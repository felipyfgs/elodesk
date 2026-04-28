<script setup lang="ts">
import { ConfirmModal } from '#components'
import type { Agent } from '~/stores/agents'
import { useAgentsStore } from '~/stores/agents'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const store = useAgentsStore()
const confirm = useOverlay().create(ConfirmModal)

const inviteOpen = ref(false)

onMounted(() => {
  store.fetch()
})

function onEdit(_agent: Agent) {
  // role change: future — for now removal handles primary lifecycle action
}

function onRemove(agent: Agent) {
  confirm.open({
    title: t('common.confirm'),
    description: t('settings.agents.removeConfirm'),
    confirmLabel: t('common.remove'),
    confirmColor: 'error',
    itemName: `${agent.name} (${agent.email})`
  }).then(async (ok) => {
    if (!ok) return
    await store.remove(agent.id)
  })
}
</script>

<template>
  <UPageCard :title="t('settings.agents.title')" variant="subtle">
    <template #footer>
      <div class="flex justify-end">
        <UButton icon="i-lucide-plus" @click="inviteOpen = true">
          {{ t('settings.agents.invite') }}
        </UButton>
      </div>
    </template>

    <SettingsAgentsTable
      :items="store.items"
      :loading="store.loading"
      @edit="onEdit"
      @remove="onRemove"
    />

    <SettingsAgentsInviteModal v-model:open="inviteOpen" />
  </UPageCard>
</template>
