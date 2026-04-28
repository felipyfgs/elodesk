<script setup lang="ts">
import type { Contact } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { useInboxesStore } from '~/stores/inboxes'

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

const selectedInboxId = ref<string | undefined>(undefined)
const selectedInbox = computed(() =>
  inboxesStore.list.find(i => i.id === selectedInboxId.value) ?? null
)

type IdentifierKind = 'phone' | 'email' | 'any'
function expectedIdentifier(channelType: string | undefined): IdentifierKind {
  if (!channelType) return 'any'
  if (channelType === 'Channel::Email') return 'email'
  if (['Channel::Whatsapp', 'Channel::Sms', 'Channel::Twilio', 'Channel::Telegram'].includes(channelType)) return 'phone'
  return 'any'
}
const identifierKind = computed(() => expectedIdentifier(selectedInbox.value?.channelType))

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

const message = ref('')
const submitting = ref(false)

interface TextareaHandle {
  inputRef?: HTMLTextAreaElement | null
  $el?: HTMLElement | null
}
const messageRef = useTemplateRef<TextareaHandle>('messageRef')

function resolveTextarea(): HTMLTextAreaElement | null {
  const r = messageRef.value
  if (!r) return null
  if (r.inputRef instanceof HTMLTextAreaElement) return r.inputRef
  const fromEl = r.$el?.querySelector?.('textarea')
  return fromEl instanceof HTMLTextAreaElement ? fromEl : null
}

function onEmojiSelect(emoji: string) {
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

watch(isOpen, async (open) => {
  if (open) {
    if (props.initialContact) setInitialContact(props.initialContact)
    else await loadRecentContacts()
    return
  }
  resetContactSearch()
  selectedInboxId.value = undefined
  message.value = ''
})

watch(contactSearchTerm, (val) => {
  startDebounce(val, isOpen.value)
})

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
        <ContactsSendMessageContactPicker
          v-model:selected-id="selectedContactId"
          :selected="selectedContact"
          :items="contactItems"
          :searching="searching"
          :can-create-from-term="canCreateFromTerm"
          :create-label="createLabel"
          :creating="creatingContact"
          @search-term="(v) => contactSearchTerm = v"
          @create-from-term="createContactFromTerm"
          @clear="selectedContactId = undefined"
        />

        <ContactsSendMessageInboxPicker
          v-model:open="isOpen"
          v-model:selected-inbox-id="selectedInboxId"
        />

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
          <ContactsSendMessageEmojiButton @select="onEmojiSelect" />
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
