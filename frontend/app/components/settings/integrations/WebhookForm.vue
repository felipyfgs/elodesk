<script setup lang="ts">
import { useWebhooksStore, type OutboundWebhook } from '~/stores/webhooks'

const props = defineProps<{ open: boolean, webhook?: OutboundWebhook | null }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const { t } = useI18n()
const toast = useToast()
const store = useWebhooksStore()

const state = reactive({ url: '', subscriptionsText: '', secret: '' })
const loading = ref(false)

watch(() => props.open, (o) => {
  if (!o) return
  if (props.webhook) {
    state.url = props.webhook.url
    state.subscriptionsText = (props.webhook.subscriptions ?? []).join(', ')
    state.secret = ''
  } else {
    state.url = ''
    state.subscriptionsText = 'message.created, conversation.updated'
    state.secret = ''
  }
})

async function onSubmit() {
  loading.value = true
  try {
    const subs = state.subscriptionsText.split(',').map(s => s.trim()).filter(Boolean)
    await store.save({
      id: props.webhook?.id,
      url: state.url,
      subscriptions: subs,
      secret: state.secret || undefined
    })
    toast.add({ title: t('common.save'), color: 'success' })
    emit('update:open', false)
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal :open="props.open" :title="props.webhook ? t('common.edit') : t('settings.integrations.newWebhook')" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-6 space-y-4">
        <UFormField label="URL">
          <UInput v-model="state.url" type="url" placeholder="https://example.com/webhook" />
        </UFormField>
        <UFormField label="Subscriptions">
          <UInput v-model="state.subscriptionsText" placeholder="message.created, conversation.updated" />
        </UFormField>
        <UFormField label="Secret (HMAC)">
          <UInput v-model="state.secret" type="password" autocomplete="off" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton variant="outline" @click="emit('update:open', false)">
            {{ t('common.cancel') }}
          </UButton>
          <UButton :loading="loading" @click="onSubmit">
            {{ t('common.save') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
