<script setup lang="ts">
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'
import type { Contact } from '~/stores/contacts'
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

// Inbox selection
const selectedInboxId = ref<string | undefined>(undefined)
const loadingInboxes = ref(false)

const selectedInbox = computed(() =>
  inboxesStore.list.find(i => i.id === selectedInboxId.value) ?? null
)

const inboxItems = computed(() =>
  inboxesStore.list.map(i => ({
    label: i.name,
    icon: channelIcon(i.channelType),
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

// Contact search composable
const {
  searchTerm: contactSearchTerm,
  searching,
  selectedId: selectedContactId,
  creating: creatingContact,
  selected: selectedContact,
  items: contactItems,
  canCreateFromTerm,
  createLabel,
  loadRecent: loadRecentContacts,
  startDebounce,
  createFromTerm: createContactFromTerm,
  reset: resetContactSearch,
  setInitial: setInitialContact
} = useContactSearch(identifierKind)

// Lifecycle
watch(isOpen, async (open) => {
  if (open) {
    await loadInboxes()
    if (props.initialContact) {
      setInitialContact(props.initialContact)
    } else {
      await loadRecentContacts()
    }
    return
  }
  resetContactSearch()
  selectedInboxId.value = undefined
  message.value = ''
})

watch(contactSearchTerm, (val) => {
  startDebounce(val, isOpen.value)
})

// Message
const message = ref('')
const submitting = ref(false)

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

interface EmojiPickerSelectEvent { i: string, n: string[], r: string, t: string, u: string }

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
            <UIcon :name="channelIcon(selectedInbox.channelType)" class="size-3.5 mr-1" />
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
          <UPopover :content="{ align: 'start', side: 'top', sideOffset: 8 }" :ui="{ content: 'p-0 overflow-hidden' }">
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
