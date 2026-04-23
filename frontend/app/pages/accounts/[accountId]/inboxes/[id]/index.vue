<script setup lang="ts">
import type { ChannelApiData, Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'
import InboxSettingsForm from '~/components/inboxes/settings/InboxSettingsForm.vue'

const props = defineProps<{
  inbox: Inbox
}>()

const emit = defineEmits<{
  inboxUpdated: [inbox: Inbox]
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const config = useRuntimeConfig()

const wa = computed(() => props.inbox.channelWhatsApp)
const apiChannel = ref<ChannelApiData | null | undefined>(props.inbox.channelApi)
const widget = computed(() => props.inbox.channelWebWidget)
const email = computed(() => props.inbox.channelEmail)
const isWhatsApp = computed(() => props.inbox.channelType === 'Channel::Whatsapp')
const isApi = computed(() => props.inbox.channelType === 'Channel::Api')
const saving = ref(false)
const formState = reactive({
  name: props.inbox.name,
  webhookUrl: props.inbox.channelApi?.webhookUrl ?? '',
  hmacMandatory: props.inbox.channelApi?.hmacMandatory ?? false
})
const publicApiBase = computed(() => {
  if (!apiChannel.value?.identifier) return ''
  const apiUrl = String(config.public.apiUrl || '').replace(/\/$/, '')
  return `${apiUrl}/public/api/v1/inboxes/${apiChannel.value.identifier}`
})

watch(() => props.inbox, (inbox) => {
  apiChannel.value = inbox.channelApi
  formState.name = inbox.name
  formState.webhookUrl = inbox.channelApi?.webhookUrl ?? ''
  formState.hmacMandatory = inbox.channelApi?.hmacMandatory ?? false
})

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

async function copy(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    toast.add({ title: t('common.copied'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  }
}

async function saveOverview() {
  if (!auth.account?.id) return
  saving.value = true
  try {
    await api(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}`, {
      method: 'PUT',
      body: { name: formState.name }
    })

    let channelApi = apiChannel.value
    if (isApi.value) {
      channelApi = await api<ChannelApiData>(`/accounts/${auth.account.id}/inboxes/api/${props.inbox.id}`, {
        method: 'PUT',
        body: {
          webhookUrl: formState.webhookUrl,
          hmacMandatory: formState.hmacMandatory,
          additionalAttributes: apiChannel.value?.additionalAttributes ?? {}
        }
      })
      apiChannel.value = channelApi
      formState.webhookUrl = channelApi.webhookUrl ?? ''
      formState.hmacMandatory = channelApi.hmacMandatory
    }

    emit('inboxUpdated', {
      ...props.inbox,
      name: formState.name,
      channelApi
    })
    toast.add({ title: t('common.success'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="grid gap-4">
    <UPageCard :title="t('inboxes.general')" variant="subtle">
      <UForm :state="formState" class="flex flex-col gap-5" @submit="saveOverview">
        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="t('inboxes.wizards.name')" name="name">
            <UInput v-model="formState.name" class="w-full" />
          </UFormField>

          <UFormField
            v-if="isApi"
            :label="t('inboxes.channelDetail.webhookUrl')"
            name="webhookUrl"
          >
            <UInput
              v-model="formState.webhookUrl"
              type="url"
              placeholder="https://example.com/webhook"
              class="w-full"
            />
          </UFormField>

          <UFormField
            v-if="isApi"
            :label="t('inboxes.channelDetail.identifier')"
            name="identifier"
            class="md:col-span-2"
          >
            <UFieldGroup>
              <UInput
                :model-value="apiChannel?.identifier ?? '-'"
                class="w-full"
                readonly
              />
              <UButton
                icon="i-lucide-copy"
                type="button"
                color="neutral"
                variant="subtle"
                :disabled="!apiChannel?.identifier"
                @click="copy(apiChannel?.identifier ?? '')"
              />
            </UFieldGroup>
          </UFormField>

          <UFormField
            v-if="isApi && publicApiBase"
            :label="t('inboxes.webhooksDetail.publicApiBase')"
            name="publicApiBase"
            class="md:col-span-2"
          >
            <UFieldGroup>
              <UInput
                :model-value="publicApiBase"
                class="w-full"
                readonly
              />
              <UButton
                icon="i-lucide-copy"
                type="button"
                color="neutral"
                variant="subtle"
                @click="copy(publicApiBase)"
              />
            </UFieldGroup>
          </UFormField>

          <UFormField v-if="isApi" name="hmacMandatory" class="md:col-span-2">
            <UCheckbox v-model="formState.hmacMandatory" :label="t('inboxes.channelDetail.hmacMandatory')" />
          </UFormField>
        </div>

        <USeparator />

        <dl class="grid gap-3 text-sm md:grid-cols-2">
          <div>
            <dt class="text-muted">
              ID
            </dt>
            <dd class="mt-1 font-mono text-xs">
              {{ props.inbox.id }}
            </dd>
          </div>

          <div>
            <dt class="text-muted">
              {{ t('common.createdAt') }}
            </dt>
            <dd class="mt-1">
              {{ formatDate(props.inbox.createdAt) }}
            </dd>
          </div>
        </dl>

        <div class="flex justify-end">
          <UButton type="submit" :loading="saving">
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UPageCard>

    <UPageCard
      v-if="inbox.channelType === 'Channel::WebWidget'"
      :title="t('inboxes.security')"
      variant="subtle"
    >
      <InboxSettingsForm :inbox="inbox" />
    </UPageCard>

    <UPageCard
      v-if="!isApi"
      :title="t('inboxes.channel')"
      variant="subtle"
    >
      <dl class="text-sm grid gap-3 md:grid-cols-2">
        <template v-if="isWhatsApp && wa">
          <div>
            <dt class="text-muted">
              {{ t('inboxes.channelDetail.provider') }}
            </dt>
            <dd class="font-medium">
              {{ wa.provider }}
            </dd>
          </div>
          <div v-if="wa.phoneNumber">
            <dt class="text-muted">
              {{ t('inboxes.channelDetail.phoneNumber') }}
            </dt>
            <dd class="font-mono text-xs">
              {{ wa.phoneNumber }}
            </dd>
          </div>
        </template>

        <template v-else-if="widget">
          <div>
            <dt class="text-muted">
              {{ t('inboxes.channelDetail.websiteUrl') }}
            </dt>
            <dd class="font-mono text-xs break-all">
              {{ widget.websiteUrl }}
            </dd>
          </div>
          <div>
            <dt class="text-muted">
              {{ t('inboxes.channelDetail.websiteToken') }}
            </dt>
            <dd class="font-mono text-xs break-all">
              {{ widget.websiteToken }}
            </dd>
          </div>
        </template>

        <template v-else-if="email">
          <div>
            <dt class="text-muted">
              {{ t('inboxes.wizards.email.email') }}
            </dt>
            <dd class="font-mono text-xs break-all">
              {{ email.email }}
            </dd>
          </div>
          <div>
            <dt class="text-muted">
              {{ t('inboxes.channelDetail.provider') }}
            </dt>
            <dd class="font-medium">
              {{ email.provider }}
            </dd>
          </div>
        </template>
      </dl>
    </UPageCard>
  </div>
</template>
