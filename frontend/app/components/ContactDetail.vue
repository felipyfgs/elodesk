<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore } from '~/stores/labels'
import { useCustomAttributesStore } from '~/stores/customAttributes'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import { contactSchema, type ContactForm } from '~/schemas/contact'
import { format } from 'date-fns'

const props = defineProps<{
  contactId: string
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const labelsStore = useLabelsStore()
const attrsStore = useCustomAttributesStore()
const convsStore = useConversationsStore()

const contact = ref<{
  id: string
  name: string | null
  email: string | null
  phone_number: string | null
  identifier: string | null
  custom_attributes: Record<string, unknown>
  labels?: { id: string }[]
} | null>(null)

const loading = ref(true)
const saving = ref(false)
const saved = ref(false)

const form = reactive<ContactForm>({
  name: '',
  email: null,
  phone_number: null,
  identifier: null
})

const activeTab = ref('info')

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const c = await api<typeof contact.value>(`/accounts/${auth.account.id}/contacts/${props.contactId}`)
    contact.value = c
    form.name = c?.name ?? ''
    form.email = c?.email
    form.phone_number = c?.phone_number
    form.identifier = c?.identifier
  } finally {
    loading.value = false
  }
}

async function saveContact(event: FormSubmitEvent<ContactForm>) {
  const accountId = auth.account?.id
  if (!accountId) return
  saving.value = true
  try {
    const updated = await api<typeof contact.value>(
      `/accounts/${accountId}/contacts/${props.contactId}`,
      { method: 'POST', body: event.data }
    )
    contact.value = updated
    saved.value = true
    setTimeout(() => {
      saved.value = false
    }, 2000)
  } finally {
    saving.value = false
  }
}

onMounted(load)

const tabs = computed(() => [
  { label: t('contactDetail.labels'), value: 'info' },
  { label: t('contactDetail.notes'), value: 'notes' },
  { label: t('contactDetail.conversations'), value: 'conversations' }
])

const contactConversations = computed(() =>
  convsStore.list.filter((c: Conversation) =>
    c.contactInbox?.contact?.name || c.contactInbox?.contact?.phoneNumber
  )
)

const _contactLabels = computed(() => labelsStore.list.filter(l =>
  contact.value?.labels?.some((cl: { id: string }) => cl.id === l.id)
))

function statusColor(status: string): 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral' {
  const map: Record<string, 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'> = { OPEN: 'success', PENDING: 'warning', RESOLVED: 'neutral', SNOOZED: 'info' }
  return map[status] ?? 'neutral'
}
</script>

<template>
  <div v-if="loading" class="flex items-center justify-center py-12 text-muted">
    {{ t('common.loading') }}
  </div>

  <div v-else-if="contact" class="space-y-6">
    <UTabs v-model="activeTab" :items="tabs" />

    <div v-if="activeTab === 'info'" class="space-y-4">
      <UPageCard variant="outline" :title="t('contactDetail.title')">
        <UAlert
          v-if="saved"
          class="mb-4"
          color="success"
          variant="subtle"
          icon="i-lucide-check-circle"
          :title="t('common.success')"
        />

        <UForm
          :schema="contactSchema"
          :state="form"
          class="flex flex-col gap-4"
          @submit="saveContact"
        >
          <UFormField :label="t('contactDetail.name')" name="name">
            <UInput v-model="form.name" class="w-full" />
          </UFormField>

          <UFormField :label="t('contactDetail.email')" name="email">
            <UInput v-model="form.email!" type="email" class="w-full" />
          </UFormField>

          <UFormField :label="t('contactDetail.phone')" name="phone_number">
            <UInput v-model="form.phone_number!" class="w-full" />
          </UFormField>

          <UFormField :label="t('contactDetail.identifier')" name="identifier">
            <UInput v-model="form.identifier!" class="w-full" />
          </UFormField>

          <div class="flex justify-end gap-2">
            <UButton type="button" variant="ghost" @click="load">
              {{ t('contactDetail.cancel') }}
            </UButton>
            <UButton type="submit" :loading="saving">
              {{ t('contactDetail.save') }}
            </UButton>
          </div>
        </UForm>
      </UPageCard>

      <UPageCard variant="outline" :title="t('contactDetail.labels')">
        <LabelPicker :contact-id="contactId" />
      </UPageCard>

      <UPageCard v-if="attrsStore.contactDefinitions.length" variant="outline" :title="t('contactDetail.attributes')">
        <div class="space-y-3">
          <CustomAttributeField
            v-for="def in attrsStore.contactDefinitions"
            :key="def.id"
            :definition="def"
            :value="(contact?.custom_attributes as Record<string, unknown>)?.[def.attributeKey]"
            :contact-id="contactId"
          />
        </div>
      </UPageCard>
    </div>

    <div v-else-if="activeTab === 'notes'">
      <NoteEditor :contact-id="contactId" />
    </div>

    <div v-else-if="activeTab === 'conversations'">
      <UPageCard variant="outline" :title="t('contactDetail.conversations')">
        <p v-if="!contactConversations.length" class="text-sm text-muted">
          {{ t('common.noResults') }}
        </p>

        <div v-else class="space-y-2">
          <div
            v-for="conv in contactConversations"
            :key="conv.id"
            class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-default"
          >
            <div class="flex items-center gap-3 min-w-0">
              <UBadge :color="statusColor(conv.status)" variant="subtle">
                {{ conv.status }}
              </UBadge>
              <div class="min-w-0">
                <p class="text-sm font-medium truncate">
                  {{ conv.inbox?.name ?? '—' }}
                </p>
                <p class="text-xs text-muted">
                  {{ format(new Date(conv.lastActivityAt), 'MMM d, HH:mm') }}
                </p>
              </div>
            </div>
            <UButton
              size="xs"
              variant="ghost"
              icon="i-lucide-message-square"
              :to="`/conversations?thread=${conv.id}`"
            />
          </div>
        </div>
      </UPageCard>
    </div>
  </div>
</template>
