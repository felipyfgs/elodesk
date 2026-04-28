<script setup lang="ts">
import type { Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useCannedResponsesStore } from '~/stores/cannedResponses'
import { useMessagesStore, type Message } from '~/stores/messages'

interface UploadedFile {
  id: string
  file: File
  url?: string
  uploading: boolean
  error?: string
}

const props = defineProps<{
  conversation: Conversation
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const canned = useCannedResponsesStore()
const messages = useMessagesStore()
const toast = useToast()

const reply = ref('')
const sending = ref(false)
const mode = ref<'reply' | 'private'>('reply')
const attachments = ref<UploadedFile[]>([])
const expanded = ref(false)
const cannedOpen = ref(false)
const isRecording = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

// Lista explícita (não `*/*`) pra evitar payloads que o channel rejeitaria.
const ACCEPT_TYPES = 'image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.odt,.ods,.odp,.txt,.csv,.rtf,.zip,.rar,.7z'

type RichEditor = {
  insertAtCursor: (text: string) => void
  replaceSlashCommand: (content: string) => void
  focus: () => void
  toggleBold: () => void
  toggleItalic: () => void
  toggleCode: () => void
  toggleBulletList: () => void
  toggleOrderedList: () => void
  undo: () => void
  redo: () => void
  setLink: (href: string) => void
  isActive: (mark: string) => boolean
}
const richEditorRef = ref<RichEditor | null>(null)

const maxChars = computed(() => {
  const channelType = props.conversation.inbox?.channelType
  switch (channelType) {
    case 'Channel::Whatsapp': return 4096
    case 'Channel::Sms': return 160
    default: return 0
  }
})

const charCount = computed(() => reply.value.length)
const charExceeded = computed(() => maxChars.value > 0 && charCount.value > maxChars.value)

const promptPlaceholder = computed(() => (
  mode.value === 'private'
    ? t('conversations.compose.privatePlaceholder')
    : t('conversations.compose.placeholder')
))

const composerShellClass = computed(() => [
  'mx-auto w-full rounded-lg px-3 py-2 shadow-lg ring transition-colors',
  expanded.value ? 'max-w-3xl flex h-[80vh] min-h-0 flex-col' : 'max-w-5xl xl:max-w-6xl 2xl:max-w-7xl',
  mode.value === 'private'
    ? 'bg-warning/5 ring-warning/25'
    : 'bg-elevated/85 ring-default'
])

function closeExpanded() {
  expanded.value = false
}

watch(expanded, (val) => {
  if (val) {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeExpanded()
        window.removeEventListener('keydown', onKey)
      }
    }
    window.addEventListener('keydown', onKey)
  }
})

const slashMatch = computed(() => {
  const m = reply.value.match(/(?:^|\s)\/(\w*)$/)
  return m ? m[1] ?? '' : null
})

const cannedSearch = computed(() => slashMatch.value ?? '')

watch(slashMatch, (val) => {
  if (val !== null && canned.list.length) {
    cannedOpen.value = true
  } else if (val === null) {
    cannedOpen.value = false
  }
})

function handleCannedSelect(content: string) {
  richEditorRef.value?.replaceSlashCommand(content)
  cannedOpen.value = false
}

function onEmojiSelect(event: { i: string }) {
  richEditorRef.value?.insertAtCursor(event.i)
}

function onRecorded(file: File) {
  isRecording.value = false
  uploadFile(file)
}

function onRecorderError(reason: 'permissionDenied' | 'unavailable' | 'unsupported') {
  isRecording.value = false
  const keys = { permissionDenied: 'voicePermissionDenied', unavailable: 'voiceUnavailable', unsupported: 'voiceUnsupported' }
  toast.add({ title: t(`conversations.compose.${keys[reason]}`), color: reason === 'unsupported' ? 'warning' : 'error' })
}

function buildInReplyTo() {
  const target = messages.replyingTo[props.conversation.id] ?? null
  if (!target) return undefined
  const isAgent = target.sender?.type === 'user' || target.senderType === 'USER'
  return { id: target.id, content: target.content ?? '', author: (isAgent ? auth.user?.name : props.conversation.meta?.sender?.name) ?? '' }
}

async function send() {
  const text = reply.value.trim()
  const readyAttachments = attachments.value.filter(a => a.url && !a.error)
  if (!auth.account?.id || (!text && !readyAttachments.length) || charExceeded.value) return
  sending.value = true

  const echoId = crypto.randomUUID()
  const isPrivate = mode.value === 'private'
  const now = new Date().toISOString()
  const inReplyTo = buildInReplyTo()
  const contentAttrs: Record<string, unknown> = { format: 'markdown', echo_id: echoId }
  if (inReplyTo) contentAttrs.in_reply_to = inReplyTo

  messages.upsert({
    id: `tmp:${echoId}`,
    echoId,
    conversationId: props.conversation.id,
    inboxId: props.conversation.inboxId,
    accountId: props.conversation.accountId,
    content: text,
    contentType: 'text',
    messageType: 1,
    senderType: 'USER',
    senderId: auth.user?.id ?? null,
    sourceId: null,
    private: isPrivate,
    status: 'sending',
    contentAttributes: contentAttrs,
    attachments: readyAttachments.map((a, idx) => ({
      id: -(idx + 1), messageId: -1, fileType: a.file.type || 'file',
      fileKey: a.url, contentType: a.file.type, size: a.file.size, createdAt: now
    })),
    createdAt: now,
    updatedAt: now
  } satisfies Message)

  try {
    const body: Record<string, unknown> = {
      message: text,
      echo_id: echoId,
      private: isPrivate,
      content_attributes: inReplyTo ? { format: 'markdown', in_reply_to: inReplyTo } : { format: 'markdown' }
    }
    if (readyAttachments.length) {
      body.attachments = readyAttachments.map(a => ({
        file_key: a.url!,
        file_name: a.file.name,
        file_type: a.file.type,
        size: a.file.size
      }))
    }
    const res = await api<Message>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`, { method: 'POST', body })
    if (res?.id) messages.upsert({ ...res, echoId: res.echoId ?? echoId })
    reply.value = ''
    messages.clearReplyTarget(props.conversation.id)
    attachments.value = []
  } catch (err) {
    messages.remove(`tmp:${echoId}`)
    throw err
  } finally {
    sending.value = false
    // Re-foca o editor após enviar para que o agente possa digitar a
    // próxima mensagem sem clicar no input. O `setContent('')` que o
    // RichTextComposer faz no watcher de modelValue (após reply.value = '')
    // remove o foco; precisamos esperar o ciclo Vue propagar a mudança
    // antes de re-focar, daí o nextTick.
    await nextTick()
    richEditorRef.value?.focus()
  }
}

async function uploadFile(file: File) {
  const id = crypto.randomUUID()
  attachments.value.push({ id, file, uploading: true })
  try {
    const form = new FormData()
    form.append('file', file, file.name)
    const res = await api<{ path: string }>(`/accounts/${auth.account!.id}/uploads`, { method: 'POST', body: form })
    const idx = attachments.value.findIndex(a => a.id === id)
    if (idx >= 0) Object.assign(attachments.value[idx]!, { url: res.path, uploading: false })
  } catch (err) {
    console.error('[Composer] upload failed', err)
    const idx = attachments.value.findIndex(a => a.id === id)
    if (idx >= 0) Object.assign(attachments.value[idx]!, { uploading: false, error: 'Upload failed' })
  }
}

function onFileInputChange(e: Event) {
  const input = e.target as HTMLInputElement
  for (const file of input.files ? Array.from(input.files) : []) uploadFile(file)
  input.value = ''
}

onMounted(async () => {
  if (!auth.account?.id || canned.list.length) return
  try {
    const res = await api<{ payload: { id: string, shortCode: string, content: string }[] }>(`/accounts/${auth.account.id}/canned_responses`)
    const now = new Date().toISOString()
    for (const item of res.payload ?? []) {
      canned.upsert({ id: item.id, accountId: auth.account!.id, shortCode: item.shortCode, content: item.content, createdAt: now, updatedAt: now })
    }
  } catch { /* ignore */ }
})
</script>

<template>
  <!--
    `pb-[max(0.375rem,env(safe-area-inset-bottom))]` reserva o espaço da
    home-indicator no iOS Safari quando o composer está colado no rodapé;
    `pb-1.5` (0.375rem) é o padding base usado em telas sem notch.
  -->
  <div
    :class="expanded
      ? 'fixed inset-0 z-40 flex flex-col items-center justify-center bg-default/70 px-4 py-6 backdrop-blur'
      : 'shrink-0 border-t border-default/80 bg-default/95 px-2 pt-1.5 pb-[max(0.375rem,env(safe-area-inset-bottom))] sm:px-4'"
    @click.self="expanded && closeExpanded()"
  >
    <ClientOnly>
      <ConversationsRichTextComposer
        ref="richEditorRef"
        v-model="reply"
        :placeholder="promptPlaceholder"
        :disabled="sending"
        :ui-class="composerShellClass"
        @submit="send"
      >
        <template #header>
          <ConversationsComposerHeader
            v-model:mode="mode"
            :conversation="conversation"
            :char-count="charCount"
            :max-chars="maxChars"
          />

          <div v-if="isRecording" class="mb-2">
            <ConversationsAudioRecorder
              @recorded="onRecorded"
              @canceled="isRecording = false"
              @error="onRecorderError"
            />
          </div>

          <ConversationsComposerAttachmentsList
            :attachments="attachments"
            @remove="(id: string) => attachments = attachments.filter(a => a.id !== id)"
          />

          <input
            ref="fileInputRef"
            type="file"
            multiple
            :accept="ACCEPT_TYPES"
            class="hidden"
            @change="onFileInputChange"
          >
        </template>

        <template #footer>
          <ConversationsComposerToolbar
            v-model:expanded="expanded"
            v-model:canned-open="cannedOpen"
            :rich-editor="richEditorRef"
            :mode="mode"
            :sending="sending"
            :disabled="charExceeded || (!reply.trim() && !attachments.length)"
            :canned-search="cannedSearch"
            @submit="send"
            @attach="fileInputRef?.click()"
            @record="isRecording = true"
            @canned-select="handleCannedSelect"
            @emoji-select="onEmojiSelect"
          />
        </template>
      </ConversationsRichTextComposer>
    </ClientOnly>
  </div>
</template>
