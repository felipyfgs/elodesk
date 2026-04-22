<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useContactsStore, type Contact } from '~/stores/contacts'
import { useLabelsStore, type Label } from '~/stores/labels'
import { useNotesStore, type Note } from '~/stores/notes'
import { useCustomAttributesStore, type CustomAttributeDefinition } from '~/stores/customAttributes'
import { useConversationsStore, type Conversation } from '~/stores/conversations'

definePageMeta({ layout: 'dashboard' })

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()
const toast = useToast()

const contactId = route.params.id as string
const loading = ref(true)
const contact = ref<Contact | null>(null)

type SidebarTab = 'attributes' | 'history' | 'notes' | 'merge'
const activeTab = ref<SidebarTab>('attributes')

const tabs = computed(() => [
  { label: t('contactDetail.sidebar.attributes'), value: 'attributes' as const, icon: 'i-lucide-braces' },
  { label: t('contactDetail.sidebar.history'), value: 'history' as const, icon: 'i-lucide-clock' },
  { label: t('contactDetail.sidebar.notes'), value: 'notes' as const, icon: 'i-lucide-sticky-note' },
  { label: t('contactDetail.sidebar.merge'), value: 'merge' as const, icon: 'i-lucide-git-merge' }
])

const breadcrumb = computed(() => [
  { label: t('contacts.title'), to: `/accounts/${auth.account?.id}/contacts`, icon: 'i-lucide-users' },
  { label: contact.value?.name || `#${contactId}` }
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

const isOwnerAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

async function toggleBlock() {
  if (!contact.value) return
  try {
    const updated = await contactsStore.setBlocked(contact.value.id, !contact.value.blocked)
    if (updated) {
      contact.value = updated
      toast.add({
        title: updated.blocked ? t('contacts.actions.blocked') : t('contacts.actions.unblocked'),
        color: 'success'
      })
    }
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    toast.add({ title: e?.response?._data?.error ?? t('common.error'), color: 'error' })
  }
}

async function sendMessage() {
  await router.push(`/accounts/${auth.account?.id}/conversations?contact=${contactId}`)
}

function onUpdated(updated: Contact) {
  contact.value = updated
}

onMounted(load)
</script>

<template>
  <UDashboardPanel id="contact-detail">
    <template #header>
      <UDashboardNavbar :ui="{ right: 'gap-2' }">
        <template #title>
          <UBreadcrumb :items="breadcrumb" />
        </template>

        <template #right>
          <UButton
            v-if="contact"
            :label="contact.blocked ? t('contacts.actions.unblock') : t('contacts.actions.block')"
            color="neutral"
            variant="outline"
            size="sm"
            :icon="contact.blocked ? 'i-lucide-shield' : 'i-lucide-shield-off'"
            @click="toggleBlock"
          />
          <UButton
            :label="t('contactDetail.sendMessage')"
            color="primary"
            size="sm"
            icon="i-lucide-message-square-plus"
            @click="sendMessage"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24 text-muted">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="contact" class="flex h-full min-h-0">
        <!-- Main column -->
        <div class="flex-1 min-w-0 overflow-y-auto">
          <div class="max-w-6xl mx-auto w-full px-6 py-6">
            <ContactsDetailContactDetailView
              :contact="contact"
              @updated="onUpdated"
            />
          </div>
        </div>

        <!-- Right sidebar -->
        <aside class="hidden xl:flex shrink-0 w-[380px] flex-col border-l border-default bg-elevated/30">
          <div class="p-3 border-b border-default">
            <UTabs
              v-model="activeTab"
              :items="tabs"
              variant="pill"
              size="xs"
              :content="false"
              class="w-full"
            />
          </div>
          <div class="flex-1 overflow-y-auto p-4">
            <ContactsDetailContactSidebarAttributes
              v-if="activeTab === 'attributes'"
              :contact="contact"
            />
            <ContactsDetailContactSidebarHistory
              v-else-if="activeTab === 'history'"
            />
            <ContactsDetailContactSidebarNotes
              v-else-if="activeTab === 'notes'"
              :contact-id="contactId"
            />
            <ContactsDetailContactSidebarMerge
              v-else-if="isOwnerAdmin"
              :contact="contact"
            />
            <p v-else class="text-sm text-muted">
              {{ t('settings.accessDenied') }}
            </p>
          </div>
        </aside>
      </div>
    </template>
  </UDashboardPanel>
</template>
