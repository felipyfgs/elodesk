<script setup lang="ts">
import WebhookList from '~/components/settings/integrations/WebhookList.vue'
import WebhookForm from '~/components/settings/integrations/WebhookForm.vue'
import IntegrationCard from '~/components/settings/integrations/IntegrationCard.vue'
import { ConfirmModal } from '#components'
import type { OutboundWebhook } from '~/stores/webhooks'
import { useWebhooksStore } from '~/stores/webhooks'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const store = useWebhooksStore()
const confirm = useOverlay().create(ConfirmModal)

const formOpen = ref(false)
const editing = ref<OutboundWebhook | null>(null)

onMounted(() => {
  store.fetch()
})

function onNew() {
  editing.value = null
  formOpen.value = true
}

function onEdit(h: OutboundWebhook) {
  editing.value = h
  formOpen.value = true
}

function onRemove(h: OutboundWebhook) {
  confirm.open({
    title: t('common.delete'),
    confirmLabel: t('common.delete'),
    itemName: h.url
  }).then(async (ok) => {
    if (!ok) return
    await store.remove(h.id)
  })
}
</script>

<template>
  <div class="space-y-6">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <IntegrationCard
        icon="i-lucide-webhook"
        :title="$t('settings.integrations.webhooks')"
        :description="'Outbound webhooks'"
        status="configured"
      />
      <IntegrationCard
        icon="i-simple-icons-slack"
        title="Slack"
        :description="'Coming soon'"
        status="soon"
      />
      <IntegrationCard
        icon="i-simple-icons-discord"
        title="Discord"
        :description="'Coming soon'"
        status="soon"
      />
    </div>

    <UPageCard :title="$t('settings.integrations.webhooks')" variant="subtle">
      <template #footer>
        <div class="flex justify-end">
          <UButton icon="i-lucide-plus" @click="onNew">
            {{ $t('settings.integrations.newWebhook') }}
          </UButton>
        </div>
      </template>

      <WebhookList
        :items="store.items"
        :loading="store.loading"
        @edit="onEdit"
        @remove="onRemove"
      />
      <WebhookForm v-model:open="formOpen" :webhook="editing" />
    </UPageCard>
  </div>
</template>
