<script setup lang="ts">
import type { ChannelApiData, Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const config = useRuntimeConfig()

const isApi = computed(() => props.inbox.channelType === 'Channel::Api')
const channelApiState = ref<ChannelApiData | null | undefined>(props.inbox.channelApi)
const formState = reactive({
  webhookUrl: props.inbox.channelApi?.webhookUrl ?? '',
  hmacMandatory: props.inbox.channelApi?.hmacMandatory ?? false
})
const saving = ref(false)

const inboundEndpoints = computed(() => {
  const apiUrl = String(config.public.apiUrl || '').replace(/\/$/, '')
  if (props.inbox.channelType === 'Channel::WebWidget' && props.inbox.channelWebWidget) {
    return [
      { label: t('inboxes.webhooksDetail.widgetScript'), value: props.inbox.channelWebWidget.embedScript }
    ]
  }
  if (props.inbox.channelType === 'Channel::Api' && channelApiState.value) {
    return [
      { label: t('inboxes.webhooksDetail.publicApiBase'), value: `${apiUrl}/public/api/v1/inboxes/${channelApiState.value.identifier}` }
    ]
  }
  return []
})

watch(() => props.inbox.channelApi, (channelApi) => {
  channelApiState.value = channelApi
  formState.webhookUrl = channelApi?.webhookUrl ?? ''
  formState.hmacMandatory = channelApi?.hmacMandatory ?? false
})

async function saveApiWebhook() {
  if (!auth.account?.id || !isApi.value) return
  saving.value = true
  try {
    const channelApi = await api<ChannelApiData>(`/accounts/${auth.account.id}/inboxes/api/${props.inbox.id}`, {
      method: 'PUT',
      body: {
        webhookUrl: formState.webhookUrl,
        hmacMandatory: formState.hmacMandatory,
        additionalAttributes: channelApiState.value?.additionalAttributes ?? {}
      }
    })
    channelApiState.value = channelApi
    toast.add({ title: t('common.success'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    saving.value = false
  }
}

async function copy(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    toast.add({ title: t('common.copied'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <UPageCard
      v-if="isApi"
      :title="t('inboxes.webhooks')"
      :description="t('inboxes.webhooksDetail.apiDescription')"
      variant="subtle"
    >
      <UForm :state="formState" class="flex flex-col gap-4" @submit="saveApiWebhook">
        <UFormField :label="t('inboxes.channelDetail.webhookUrl')" name="webhookUrl">
          <UInput
            v-model="formState.webhookUrl"
            type="url"
            placeholder="https://example.com/webhook"
            class="w-full"
          />
        </UFormField>

        <UFormField name="hmacMandatory">
          <UCheckbox v-model="formState.hmacMandatory" :label="t('inboxes.channelDetail.hmacMandatory')" />
        </UFormField>

        <div class="flex justify-end">
          <UButton :loading="saving" type="submit">
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UPageCard>

    <UPageCard
      :title="t('inboxes.webhooksDetail.inboundTitle')"
      :description="t('inboxes.webhooksDetail.inboundDescription')"
      variant="subtle"
    >
      <div v-if="inboundEndpoints.length" class="flex flex-col gap-3">
        <div
          v-for="endpoint in inboundEndpoints"
          :key="endpoint.label"
          class="rounded-lg bg-elevated p-3"
        >
          <div class="text-xs font-medium text-muted">
            {{ endpoint.label }}
          </div>
          <div class="mt-2 flex items-start gap-2">
            <code class="font-mono text-xs break-all">{{ endpoint.value }}</code>
            <UButton
              icon="i-lucide-copy"
              variant="ghost"
              size="xs"
              color="neutral"
              @click="copy(endpoint.value)"
            />
          </div>
        </div>
      </div>

      <UAlert
        v-else
        color="neutral"
        variant="subtle"
        icon="i-lucide-webhook"
        :title="t('inboxes.webhooksDetail.providerManaged')"
        :description="t('inboxes.webhooksDetail.providerManagedDescription')"
      />
    </UPageCard>
  </div>
</template>
