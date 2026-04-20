<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useNotesStore, type Note } from '~/stores/notes'
import { useCustomAttributesStore, type CustomAttributeDefinition } from '~/stores/customAttributes'
import { useConversationsStore, type Conversation } from '~/stores/conversations'
import type { Contact } from '~/stores/contacts'

definePageMeta({ layout: 'dashboard' })

const route = useRoute()
const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const contactId = route.params.id as string
const loading = ref(true)

const contact = ref<Contact | null>(null)

const tabs = computed(() => [
  { label: t('contacts.tabs.overview'), to: `/contacts/${contactId}` },
  { label: t('contacts.tabs.conversations'), to: `/contacts/${contactId}/conversations` },
  { label: t('contacts.tabs.notes'), to: `/contacts/${contactId}/notes` },
  { label: t('contacts.tabs.events'), to: `/contacts/${contactId}/events` }
])

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const [c, labels, notes, attrs, convs] = await Promise.all([
      api<Contact>(`/accounts/${auth.account.id}/contacts/${contactId}`),
      api<Label[]>(`/accounts/${auth.account.id}/contacts/${contactId}/labels`).catch(() => []),
      api<Note[]>(`/accounts/${auth.account.id}/contacts/${contactId}/notes`).catch(() => []),
      api<CustomAttributeDefinition[]>(`/accounts/${auth.account.id}/custom_attribute_definitions`).catch(() => []),
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
          <UButton
            icon="i-lucide-arrow-left"
            color="neutral"
            variant="ghost"
            to="/contacts"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24 text-muted">
        {{ t('common.loading') }}
      </div>

      <template v-else-if="contact">
        <ContactsDetailContactHeader :contact="contact" />

        <UDashboardToolbar class="my-4">
          <UNavigationMenu
            :items="tabs"
            highlight
          />
        </UDashboardToolbar>

        <NuxtPage :contact="contact" :contact-id="contactId" />
      </template>
    </template>
  </UDashboardPanel>
</template>
