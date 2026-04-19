<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Label } from '~/stores/labels'
import type { Note } from '~/stores/notes'
import type { CustomAttributeDefinition } from '~/stores/customAttributes'
import type { Conversation } from '~/stores/conversations'

const route = useRoute()
const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const contactId = route.params.id as string
const loading = ref(true)

const contact = ref<{
  id: string
  name: string | null
  email: string | null
  phone_number: string | null
  identifier: string | null
  custom_attributes: Record<string, unknown>
  labels: Label[]
} | null>(null)

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const [c, labels, notes, attrs, convs] = await Promise.all([
      api<typeof contact.value>(`/accounts/${auth.account.id}/contacts/${contactId}`),
      api<Label[]>(`/accounts/${auth.account.id}/contacts/${contactId}/labels`).catch(() => []),
      api<Note[]>(`/accounts/${auth.account.id}/contacts/${contactId}/notes`).catch(() => []),
      api<CustomAttributeDefinition[]>(`/accounts/${auth.account.id}/custom_attributes?model=contact`).catch(() => []),
      api<Conversation[]>(`/accounts/${auth.account.id}/contacts/${contactId}/conversations`).catch(() => [])
    ])

    contact.value = c
    useLabelsStore().setAll(labels)
    useNotesStore().setForContact(contactId, notes)
    useCustomAttributesStore().setAll(attrs)
    useConversationsStore().setAll(convs)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <UDashboardPanel id="contact-detail">
    <template #header>
      <UDashboardNavbar :title="t('contactDetail.title')">
        <template #leading>
          <UButton icon="i-lucide-arrow-left" color="neutral" variant="ghost" to="/contacts" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24 text-muted">
        {{ t('common.loading') }}
      </div>

      <ContactDetail v-else-if="contact" :contact-id="contactId" />
    </template>
  </UDashboardPanel>
</template>
