<script setup lang="ts">
import type { Message, ForwardTarget } from '~/stores/messages'
import { useMessagesStore } from '~/stores/messages'
import { useConversationsStore } from '~/stores/conversations'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'
import { getAttachments, isInboxCompatibleWithAttachments } from '~/utils/chatAdapter'
import type { Contact } from '~/stores/contacts'

const model = defineModel<boolean>('open', { required: true })

const props = defineProps<{
  messageIds: string[]
  selectedMessages: Message[]
}>()

const emit = defineEmits<{ done: [] }>()

const { t } = useI18n()
const toast = useToast()
const messagesStore = useMessagesStore()
const conversations = useConversationsStore()
const inboxes = useInboxesStore()
const auth = useAuthStore()
const api = useApi()

const selectedTargets = ref<ForwardTarget[]>([])
const searchQuery = ref('')
const expandedContactId = ref<string | null>(null)

// Default contacts list (loaded once when the modal opens). Replaced by the
// search-API result while the user is typing so contacts without conversations
// are first-class targets — same as WhatsApp's forward picker.
const defaultContacts = ref<Contact[]>([])
const searchResults = ref<Contact[]>([])
const searching = ref(false)
const loadingContacts = ref(false)
const initialLoaded = ref(false)

const inboxesList = computed(() => inboxes.list)

const messageFileTypes = computed(() => {
  const types = new Set<string>()
  for (const msg of props.selectedMessages) {
    for (const att of getAttachments(msg)) {
      types.add(att.fileType)
    }
  }
  return types
})

function inboxCompatible(channelType: string): boolean {
  if (messageFileTypes.value.size === 0) return true
  return isInboxCompatibleWithAttachments(channelType, props.selectedMessages)
}

const trimmedQuery = computed(() => searchQuery.value.trim().toLowerCase())

// Filter recent conversations by the search query (client-side) so the same
// search box drives both sections. Keeps the recent-first ordering so users
// can still scan top contacts at a glance.
const filteredConversations = computed(() => {
  const all = [...conversations.list].sort(
    (a, b) => new Date(b.lastActivityAt ?? 0).getTime() - new Date(a.lastActivityAt ?? 0).getTime()
  )
  if (!trimmedQuery.value) return all.slice(0, 20)
  return all.filter((c) => {
    const name = c.meta?.sender?.name?.toLowerCase() ?? ''
    const phone = c.meta?.sender?.phoneNumber?.toLowerCase() ?? ''
    const email = c.meta?.sender?.email?.toLowerCase() ?? ''
    return name.includes(trimmedQuery.value)
      || phone.includes(trimmedQuery.value)
      || email.includes(trimmedQuery.value)
  }).slice(0, 20)
})

// Visible contacts: search results when the user is typing, otherwise the
// default list loaded on open.
const visibleContacts = computed<Contact[]>(() => {
  return trimmedQuery.value ? searchResults.value : defaultContacts.value
})

async function fetchDefaultContacts() {
  if (!auth.account?.id || initialLoaded.value) return
  loadingContacts.value = true
  try {
    const res = await api<{ payload: Contact[] }>(
      `/accounts/${auth.account.id}/contacts?pageSize=50`
    )
    defaultContacts.value = res.payload ?? []
    initialLoaded.value = true
  } catch (err) {
    if (import.meta.dev) console.warn('[ForwardModal] failed to load contacts', err)
  } finally {
    loadingContacts.value = false
  }
}

// The modal can be opened from pages that haven't loaded the inboxes store
// (the conversations page does, but ad-hoc entry points may not). Fetch on
// demand so the per-contact inbox picker is never empty. The endpoint
// returns the array directly (useApi unwraps the success/data envelope).
const inboxesLoading = ref(false)
async function ensureInboxesLoaded() {
  if (!auth.account?.id || inboxes.list.length > 0) return
  inboxesLoading.value = true
  try {
    const res = await api<Inbox[]>(`/accounts/${auth.account.id}/inboxes`)
    if (Array.isArray(res)) inboxes.setAll(res)
  } catch (err) {
    if (import.meta.dev) console.warn('[ForwardModal] failed to load inboxes', err)
  } finally {
    inboxesLoading.value = false
  }
}

async function searchContacts(query: string): Promise<Contact[]> {
  if (!query.trim() || !auth.account?.id) return []
  try {
    const res = await api<{ payload: Contact[] }>(
      `/accounts/${auth.account.id}/contacts?search=${encodeURIComponent(query)}&pageSize=20`
    )
    return res.payload ?? []
  } catch {
    return []
  }
}

watchDebounced(searchQuery, async (val) => {
  if (!val.trim()) {
    searchResults.value = []
    return
  }
  searching.value = true
  searchResults.value = await searchContacts(val)
  searching.value = false
}, { debounce: 300 })

watch(model, (open) => {
  if (open) {
    fetchDefaultContacts()
    ensureInboxesLoaded()
  } else {
    selectedTargets.value = []
    searchQuery.value = ''
    searchResults.value = []
    expandedContactId.value = null
  }
})

function addTarget(target: ForwardTarget) {
  if (selectedTargets.value.length >= 5) {
    toast.add({ title: t('conversations.forward.modal.maxTargets'), color: 'warning' })
    return
  }
  const exists = selectedTargets.value.some((existing) => {
    if ('conversationId' in existing && 'conversationId' in target) {
      return existing.conversationId === target.conversationId
    }
    if ('contactId' in existing && 'contactId' in target) {
      return existing.contactId === target.contactId
        && 'inboxId' in existing && 'inboxId' in target
        && existing.inboxId === target.inboxId
    }
    return false
  })
  if (exists) return
  selectedTargets.value.push(target)
}

function removeTarget(index: number) {
  selectedTargets.value.splice(index, 1)
}

function contactDisplayName(c: Contact): string {
  return c.name ?? c.email ?? c.phoneNumber ?? `#${c.id}`
}

function submit() {
  if (selectedTargets.value.length === 0) return
  // Fire-and-forget: close the modal immediately and let the request run in
  // the background. Toasts surface the outcome when it resolves so the user
  // isn't blocked waiting for every fan-out send to finish.
  const sourceMessageIds = props.messageIds
  const targets = [...selectedTargets.value]
  emit('done')
  messagesStore.forward({ sourceMessageIds, targets })
    .then((result) => {
      const successCount = result.results?.filter(r => r.status === 'success').length ?? 0
      if (successCount > 0) {
        toast.add({ title: t('conversations.forward.success', { count: successCount }), color: 'success' })
      }
      const failCount = result.results?.filter(r => r.status === 'failed').length ?? 0
      if (failCount > 0) {
        toast.add({
          title: t('conversations.forward.partialFailure', { ok: successCount, total: result.results.length }),
          color: 'warning'
        })
      }
    })
    .catch((err: unknown) => {
      console.error('[ForwardModal] forward failed', err)
      toast.add({ title: t('conversations.forward.failed'), color: 'error' })
    })
}
</script>

<template>
  <UModal v-model:open="model" :title="t('conversations.forward.modal.title', { count: messageIds.length })">
    <template #body>
      <div class="flex flex-col gap-3">
        <UInput
          v-model="searchQuery"
          :placeholder="t('conversations.forward.modal.searchPlaceholder')"
          size="lg"
        >
          <template #leading>
            <UIcon name="i-lucide-search" class="size-4 text-dimmed" />
          </template>
          <template #trailing>
            <UIcon v-if="searching" name="i-lucide-loader-circle" class="size-4 animate-spin text-dimmed" />
          </template>
        </UInput>

        <div class="flex flex-col gap-3 max-h-96 overflow-y-auto">
          <!-- Recent conversations -->
          <div v-if="filteredConversations.length > 0" class="flex flex-col gap-1">
            <p class="text-xs font-medium text-muted">
              {{ t('conversations.forward.modal.recentConversations') }}
            </p>
            <div class="flex flex-col gap-0.5">
              <button
                v-for="conv in filteredConversations"
                :key="'conv:' + conv.id"
                :disabled="!inboxCompatible(conv.inbox?.channelType ?? '')"
                class="flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors"
                :class="[
                  inboxCompatible(conv.inbox?.channelType ?? '')
                    ? 'hover:bg-elevated cursor-pointer'
                    : 'opacity-40 cursor-not-allowed'
                ]"
                @click="addTarget({ conversationId: String(conv.id) })"
              >
                <ConversationsContactAvatar
                  :name="conv.meta?.sender?.name ?? ''"
                  :url="conv.meta?.sender?.thumbnail ?? undefined"
                  size="sm"
                />
                <span class="flex-1 truncate">{{ conv.meta?.sender?.name ?? `#${conv.displayId ?? conv.id}` }}</span>
                <UBadge
                  v-if="conv.inbox?.name"
                  :label="conv.inbox.name"
                  color="neutral"
                  variant="soft"
                  size="xs"
                />
              </button>
            </div>
          </div>

          <!-- Contacts (always visible — contacts without conversations are
               valid forward targets, the user just picks an inbox). -->
          <div class="flex flex-col gap-1">
            <p class="text-xs font-medium text-muted">
              {{ t('conversations.forward.modal.contacts') }}
            </p>
            <div v-if="loadingContacts && visibleContacts.length === 0" class="py-4 text-center text-sm text-muted">
              <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin inline mr-2" />
              {{ t('common.loading') }}
            </div>
            <div v-else-if="visibleContacts.length === 0" class="py-4 text-center text-sm text-muted">
              {{ t('common.noResults') }}
            </div>
            <div v-else class="flex flex-col gap-0.5">
              <template v-for="contact in visibleContacts" :key="'contact:' + contact.id">
                <button
                  class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-elevated cursor-pointer"
                  @click="expandedContactId = expandedContactId === contact.id ? null : contact.id"
                >
                  <ConversationsContactAvatar
                    :name="contact.name ?? ''"
                    :url="contact.thumbnail ?? undefined"
                    size="sm"
                  />
                  <div class="flex-1 min-w-0">
                    <div class="truncate">
                      {{ contactDisplayName(contact) }}
                    </div>
                    <div v-if="contact.phoneNumber || contact.email" class="truncate text-xs text-muted">
                      {{ contact.phoneNumber ?? contact.email }}
                    </div>
                  </div>
                  <UIcon
                    :name="expandedContactId === contact.id ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                    class="size-4 text-dimmed"
                  />
                </button>

                <div v-if="expandedContactId === contact.id" class="ml-6 flex flex-col gap-0.5 border-l border-default pl-2">
                  <!-- Always show every account inbox; the backend creates the
                       contact_inbox + conversation on demand when needed. The
                       user picks how to reach the contact, the system handles
                       the rest. -->
                  <p v-if="inboxesLoading && inboxesList.length === 0" class="px-2 py-1 text-xs text-muted">
                    <UIcon name="i-lucide-loader-circle" class="size-3 animate-spin inline mr-1" />
                    {{ t('common.loading') }}
                  </p>
                  <p v-else-if="inboxesList.length === 0" class="px-2 py-1 text-xs text-muted">
                    {{ t('common.noResults') }}
                  </p>
                  <button
                    v-for="inbox in inboxesList"
                    :key="'inbox:' + contact.id + ':' + inbox.id"
                    :disabled="!inboxCompatible(inbox.channelType)"
                    class="flex items-center gap-2 rounded-md px-2 py-1 text-left text-xs transition-colors"
                    :class="[
                      inboxCompatible(inbox.channelType)
                        ? 'hover:bg-elevated cursor-pointer'
                        : 'opacity-40 cursor-not-allowed'
                    ]"
                    @click="addTarget({ contactId: contact.id, inboxId: inbox.id })"
                  >
                    <UIcon name="i-lucide-inbox" class="size-3 text-dimmed" />
                    <span class="flex-1 truncate">{{ inbox.name }}</span>
                    <UBadge
                      :label="inbox.channelType.replace('Channel::', '')"
                      color="neutral"
                      variant="soft"
                      size="xs"
                    />
                  </button>
                </div>
              </template>
            </div>
          </div>
        </div>

        <!-- Selected targets chips -->
        <div v-if="selectedTargets.length > 0" class="flex flex-wrap gap-1">
          <UBadge
            v-for="(target, i) in selectedTargets"
            :key="i"
            color="primary"
            variant="solid"
            size="sm"
          >
            <span class="mr-1">
              {{ 'conversationId' in target ? `Conversa #${target.conversationId}` : `Contato #${target.contactId}` }}
            </span>
            <button class="ml-1 hover:opacity-70" @click="removeTarget(i)">
              <UIcon name="i-lucide-x" class="size-3" />
            </button>
          </UBadge>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <span class="text-xs text-muted">{{ selectedTargets.length }}/5</span>
        <div class="flex items-center gap-2">
          <UButton color="neutral" variant="ghost" @click="model = false">
            {{ t('conversations.forward.cancel') }}
          </UButton>
          <UButton
            :disabled="selectedTargets.length === 0"
            color="primary"
            @click="submit"
          >
            {{ t('conversations.forward.submitButton', { count: selectedTargets.length }) }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
