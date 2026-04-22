<script setup lang="ts">
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'
import type { Contact, ContactListResponse } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore, type Inbox } from '~/stores/inboxes'

const props = defineProps<{
  open: boolean
  initialContact?: Contact | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'sent': [conversation: { id: string, displayId: number }]
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const inboxesStore = useInboxesStore()
const errorHandler = useErrorHandler()

const isOpen = computed({
  get: () => props.open,
  set: v => emit('update:open', v)
})

// --- Inbox select --------------------------------------------------------

const selectedInboxId = ref<string | undefined>(undefined)
const loadingInboxes = ref(false)

const selectedInbox = computed(() =>
  inboxesStore.list.find(i => i.id === selectedInboxId.value) ?? null
)

function inboxIcon(channelType: string): string {
  const map: Record<string, string> = {
    'Channel::Api': 'i-lucide-webhook',
    'Channel::Whatsapp': 'i-simple-icons-whatsapp',
    'Channel::Twilio': 'i-lucide-cloud',
    'Channel::Sms': 'i-lucide-message-square',
    'Channel::Instagram': 'i-simple-icons-instagram',
    'Channel::FacebookPage': 'i-simple-icons-facebook',
    'Channel::Telegram': 'i-simple-icons-telegram',
    'Channel::Line': 'i-simple-icons-line',
    'Channel::Tiktok': 'i-simple-icons-tiktok',
    'Channel::WebWidget': 'i-lucide-globe',
    'Channel::Email': 'i-lucide-mail',
    'Channel::Twitter': 'i-simple-icons-x'
  }
  return map[channelType] ?? 'i-lucide-inbox'
}

const inboxItems = computed(() =>
  inboxesStore.list.map(i => ({
    label: i.name,
    icon: inboxIcon(i.channelType),
    value: i.id
  }))
)

type IdentifierKind = 'phone' | 'email' | 'any'

function expectedIdentifier(channelType: string | undefined): IdentifierKind {
  if (!channelType) return 'any'
  if (channelType === 'Channel::Email') return 'email'
  if (['Channel::Whatsapp', 'Channel::Sms', 'Channel::Twilio', 'Channel::Telegram'].includes(channelType)) return 'phone'
  return 'any'
}

const identifierKind = computed(() => expectedIdentifier(selectedInbox.value?.channelType))

async function loadInboxes() {
  if (inboxesStore.list.length > 0 || !auth.account?.id) return
  loadingInboxes.value = true
  try {
    const res = await api<Inbox[]>(`/accounts/${auth.account.id}/inboxes`)
    inboxesStore.setAll(res)
  } catch (err) {
    errorHandler.handle(err)
  } finally {
    loadingInboxes.value = false
  }
}

// --- Contact search + create -------------------------------------------

const contactSearchTerm = ref('')
const contactResults = ref<Contact[]>([])
const searching = ref(false)
const selectedContactId = ref<string | undefined>(undefined)
const creatingContact = ref(false)

const selectedContact = computed(() =>
  contactResults.value.find(c => c.id === selectedContactId.value) ?? null
)

// Monotonic sequence id to drop stale responses (last-wins).
let searchSeq = 0

async function runContactSearch(term: string) {
  if (!auth.account?.id) return
  const seq = ++searchSeq
  searching.value = true
  try {
    const qs = term ? `?search=${encodeURIComponent(term)}&pageSize=20` : `?pageSize=30`
    const res = await api<ContactListResponse>(`/accounts/${auth.account.id}/contacts${qs}`)
    // Only apply if no newer request has started
    if (seq === searchSeq) contactResults.value = res.payload
  } catch (err) {
    if (seq === searchSeq) errorHandler.handle(err)
  } finally {
    if (seq === searchSeq) searching.value = false
  }
}

async function loadRecentContacts() {
  await runContactSearch('')
}

let debounceTimer: ReturnType<typeof setTimeout> | null = null

function clearDebounce() {
  if (debounceTimer) {
    clearTimeout(debounceTimer)
    debounceTimer = null
  }
}

watch(contactSearchTerm, (val) => {
  clearDebounce()
  // Only search when modal is open — avoids a stray fetch on close-reset.
  if (!isOpen.value) return
  debounceTimer = setTimeout(() => runContactSearch(val), 300)
})

onUnmounted(clearDebounce)

const contactItems = computed(() =>
  contactResults.value.map(c => ({
    label: c.name ?? c.email ?? c.phoneNumber ?? c.identifier ?? `#${c.id}`,
    description: [c.phoneNumber, c.email].filter(Boolean).join(' · '),
    icon: 'i-lucide-user',
    value: c.id
  }))
)

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const PHONE_RE = /^\+?[\d\s\-()]{6,}$/

function detectKind(value: string): IdentifierKind {
  const v = value.trim()
  if (!v) return 'any'
  if (EMAIL_RE.test(v)) return 'email'
  if (PHONE_RE.test(v)) return 'phone'
  return 'any'
}

const canCreateFromTerm = computed(() => {
  const term = contactSearchTerm.value.trim()
  if (!term || searching.value || creatingContact.value || contactResults.value.length > 0) return false
  const detected = detectKind(term)
  const expected = identifierKind.value
  if (expected === 'any') return detected !== 'any' || term.length >= 3
  return detected === expected
})

const createLabel = computed(() => {
  const term = contactSearchTerm.value.trim()
  const detected = detectKind(term)
  if (detected === 'email') return t('contactsSendMessage.createWithEmail', { value: term })
  if (detected === 'phone') return t('contactsSendMessage.createWithPhone', { value: term })
  return t('contactsSendMessage.createWith', { value: term })
})

async function createContactFromTerm() {
  const term = contactSearchTerm.value.trim()
  if (!term || !auth.account?.id) return
  creatingContact.value = true
  try {
    const kind = detectKind(term)
    const payload: Record<string, string> = {}
    if (kind === 'email') {
      payload.email = term
      payload.name = term.split('@')[0] ?? term
    } else if (kind === 'phone') {
      payload.phone_number = term
      payload.name = term
    } else {
      payload.identifier = term
      payload.name = term
    }
    const created = await api<Contact>(`/accounts/${auth.account.id}/contacts`, {
      method: 'POST',
      body: payload
    })
    contactResults.value = [created, ...contactResults.value]
    selectedContactId.value = created.id
    contactSearchTerm.value = ''
  } catch (err) {
    errorHandler.handle(err, { title: t('contactsSendMessage.createFailed') })
  } finally {
    creatingContact.value = false
  }
}

// --- Lifecycle ----------------------------------------------------------

watch(isOpen, async (open) => {
  if (open) {
    await loadInboxes()
    if (props.initialContact) {
      const initial = props.initialContact
      // Dedup: replace if already present, else prepend
      const rest = contactResults.value.filter(c => c.id !== initial.id)
      contactResults.value = [initial, ...rest]
      selectedContactId.value = initial.id
    } else {
      await loadRecentContacts()
    }
    return
  }
  // Cancel in-flight/pending search and reset form
  clearDebounce()
  searchSeq++
  contactSearchTerm.value = ''
  contactResults.value = []
  selectedContactId.value = undefined
  selectedInboxId.value = undefined
  message.value = ''
})

// --- Message ------------------------------------------------------------

const message = ref('')
const submitting = ref(false)

// Nuxt UI UTextarea exposes its internal `<textarea>` via `inputRef`.
// Fall back to querying `$el` for resilience across versions.
interface TextareaHandle {
  inputRef?: HTMLTextAreaElement | null
  $el?: HTMLElement | null
}
const messageRef = useTemplateRef<TextareaHandle>('messageRef')

function resolveTextarea(): HTMLTextAreaElement | null {
  const ref = messageRef.value
  if (!ref) return null
  if (ref.inputRef instanceof HTMLTextAreaElement) return ref.inputRef
  const fromEl = ref.$el?.querySelector?.('textarea')
  return fromEl instanceof HTMLTextAreaElement ? fromEl : null
}

interface EmojiPickerSelectEvent {
  i: string
  n: string[]
  r: string
  t: string
  u: string
}

const colorMode = useColorMode()
const emojiTheme = computed<'dark' | 'light'>(() =>
  colorMode.value === 'dark' ? 'dark' : 'light'
)

const emojiStaticTexts = computed(() => ({
  placeholder: t('contactsSendMessage.emoji.searchPlaceholder')
}))

const emojiGroupNames = computed(() => ({
  smileys_people: t('contactsSendMessage.emoji.groups.smileysPeople'),
  animals_nature: t('contactsSendMessage.emoji.groups.animalsNature'),
  food_drink: t('contactsSendMessage.emoji.groups.foodDrink'),
  activities: t('contactsSendMessage.emoji.groups.activities'),
  travel_places: t('contactsSendMessage.emoji.groups.travelPlaces'),
  objects: t('contactsSendMessage.emoji.groups.objects'),
  symbols: t('contactsSendMessage.emoji.groups.symbols'),
  flags: t('contactsSendMessage.emoji.groups.flags'),
  recent: t('contactsSendMessage.emoji.groups.recent')
}))

function onEmojiSelect(event: EmojiPickerSelectEvent) {
  const emoji = event.i
  const ta = resolveTextarea()
  if (!ta) {
    message.value += emoji
    return
  }
  const start = ta.selectionStart ?? message.value.length
  const end = ta.selectionEnd ?? message.value.length
  message.value = message.value.slice(0, start) + emoji + message.value.slice(end)
  nextTick(() => {
    ta.focus()
    const pos = start + emoji.length
    ta.setSelectionRange(pos, pos)
  })
}

function clearContact() {
  selectedContactId.value = undefined
}

function clearInbox() {
  selectedInboxId.value = undefined
}

interface ConversationResp { id: string | number, displayId: number }

async function onSubmit() {
  if (!auth.account?.id || !selectedContact.value || !selectedInbox.value) return
  const trimmed = message.value.trim()
  if (!trimmed) return

  submitting.value = true
  try {
    const res = await api<ConversationResp>(
      `/accounts/${auth.account.id}/conversations`,
      {
        method: 'POST',
        body: {
          contact_id: Number(selectedContact.value.id),
          inbox_id: Number(selectedInbox.value.id),
          message: { content: trimmed }
        }
      }
    )
    errorHandler.success(t('contactsSendMessage.success'))
    emit('sent', { id: String(res.id), displayId: res.displayId })
    isOpen.value = false
    await navigateTo(`/accounts/${auth.account.id}/conversations/${res.displayId}`)
  } catch (err) {
    errorHandler.handle(err, { title: t('contactsSendMessage.failed') })
  } finally {
    submitting.value = false
  }
}

const canSubmit = computed(
  () => !submitting.value
    && !!selectedContactId.value
    && !!selectedInboxId.value
    && message.value.trim().length > 0
)
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="t('contactsSendMessage.title')"
    :ui="{ content: 'sm:max-w-xl', body: 'p-0 sm:p-0' }"
  >
    <template #body>
      <div class="divide-y divide-default">
        <!-- Para: contato -->
        <div class="flex items-center gap-3 px-4 py-2.5 min-h-11">
          <span class="text-sm font-medium text-muted shrink-0 w-10">
            {{ t('contactsSendMessage.toLabel') }}:
          </span>
          <UBadge
            v-if="selectedContact"
            color="primary"
            variant="soft"
            size="md"
          >
            <span class="truncate max-w-[14rem]">
              {{ selectedContact.name ?? selectedContact.email ?? selectedContact.phoneNumber ?? '—' }}
            </span>
            <UButton
              icon="i-lucide-x"
              variant="link"
              color="neutral"
              size="xs"
              :padded="false"
              class="ml-1 p-0"
              @click="clearContact"
            />
          </UBadge>
          <USelectMenu
            v-else
            v-model="selectedContactId"
            :items="contactItems"
            value-key="value"
            searchable
            :searchable-placeholder="t('contactsSendMessage.searchContactPlaceholder')"
            :placeholder="t('contactsSendMessage.toPlaceholder')"
            :loading="searching"
            variant="ghost"
            class="flex-1"
            @update:search-term="contactSearchTerm = $event"
          >
            <template #empty="{ searchTerm }">
              <div v-if="canCreateFromTerm" class="p-1">
                <UButton
                  :label="createLabel"
                  :loading="creatingContact"
                  icon="i-lucide-plus"
                  color="primary"
                  variant="soft"
                  block
                  @click="createContactFromTerm"
                />
              </div>
              <p v-else class="text-sm text-muted text-center py-3 px-2">
                {{ searchTerm ? t('contactsSendMessage.noMatch') : t('contactsSendMessage.typeToSearch') }}
              </p>
            </template>
          </USelectMenu>
        </div>

        <!-- Via: inbox -->
        <div class="flex items-center gap-3 px-4 py-2.5 min-h-11">
          <span class="text-sm font-medium text-muted shrink-0 w-10">
            {{ t('contactsSendMessage.viaLabel') }}:
          </span>
          <UBadge
            v-if="selectedInbox"
            color="primary"
            variant="soft"
            size="md"
          >
            <UIcon :name="inboxIcon(selectedInbox.channelType)" class="size-3.5 mr-1" />
            <span class="truncate max-w-[14rem]">{{ selectedInbox.name }}</span>
            <UButton
              icon="i-lucide-x"
              variant="link"
              color="neutral"
              size="xs"
              :padded="false"
              class="ml-1 p-0"
              @click="clearInbox"
            />
          </UBadge>
          <USelectMenu
            v-else
            v-model="selectedInboxId"
            :items="inboxItems"
            value-key="value"
            :placeholder="t('contactsSendMessage.viaPlaceholder')"
            :loading="loadingInboxes"
            variant="ghost"
            class="flex-1"
          />
        </div>

        <!-- Mensagem -->
        <div class="px-4 py-2">
          <UTextarea
            ref="messageRef"
            v-model="message"
            :placeholder="t('contactsSendMessage.messagePlaceholder')"
            :rows="6"
            autoresize
            variant="none"
            class="w-full"
            :ui="{ base: 'px-0 py-1 min-h-36' }"
          />
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between gap-2 w-full">
        <div class="flex items-center gap-1">
          <UPopover
            :content="{ align: 'start', side: 'top', sideOffset: 8 }"
            :ui="{ content: 'p-0 overflow-hidden' }"
          >
            <UButton
              icon="i-lucide-smile"
              color="neutral"
              variant="ghost"
              size="sm"
              square
              :aria-label="t('contactsSendMessage.emojiAria')"
            />
            <template #content>
              <ClientOnly>
                <EmojiPicker
                  :native="true"
                  :hide-search="false"
                  :disable-sticky-group-names="true"
                  :disable-skin-tones="true"
                  :theme="emojiTheme"
                  :static-texts="emojiStaticTexts"
                  :group-names="emojiGroupNames"
                  @select="onEmojiSelect"
                />
              </ClientOnly>
            </template>
          </UPopover>
        </div>

        <div class="flex items-center gap-2">
          <UButton
            :label="t('contactsSendMessage.discard')"
            color="neutral"
            variant="ghost"
            :disabled="submitting"
            @click="isOpen = false"
          />
          <UButton
            :label="t('contactsSendMessage.send')"
            icon="i-lucide-send"
            :loading="submitting"
            :disabled="!canSubmit"
            @click="onSubmit"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
