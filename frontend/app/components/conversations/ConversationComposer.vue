<script setup lang="ts">
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'
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

interface EmojiPickerSelectEvent { i: string, n: string[], r: string, t: string, u: string }

const props = defineProps<{
  conversation: Conversation
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const canned = useCannedResponsesStore()
const messages = useMessagesStore()
const colorMode = useColorMode()

const reply = ref('')
const sending = ref(false)
const mode = ref<'reply' | 'private'>('reply')
const attachments = ref<UploadedFile[]>([])
const selectedFiles = ref<File[]>([])
const fileUploadRef = ref<HTMLElement | null>(null)

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

const cannedOpen = ref(false)
const emojiOpen = ref(false)
const expanded = ref(false)

const isRecording = ref(false)
const toast = useToast()

function startRecording() {
  isRecording.value = true
}

function onRecorded(file: File) {
  isRecording.value = false
  uploadFile(file)
  selectedFiles.value = [...selectedFiles.value, file]
}

function onRecorderCanceled() {
  isRecording.value = false
}

function onRecorderError(reason: 'permissionDenied' | 'unavailable' | 'unsupported') {
  isRecording.value = false
  const key = reason === 'permissionDenied'
    ? 'conversations.compose.voicePermissionDenied'
    : reason === 'unavailable'
      ? 'conversations.compose.voiceUnavailable'
      : 'conversations.compose.voiceUnsupported'
  toast.add({ title: t(key), color: reason === 'unsupported' ? 'warning' : 'error' })
}

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

function promptLink() {
  const href = window.prompt(t('conversations.compose.linkPrompt'), '')
  if (href === null) return
  richEditorRef.value?.setLink(href)
}

const composerShellClass = computed(() => [
  'mx-auto w-full rounded-lg px-3 py-2 shadow-lg ring transition-colors',
  expanded.value ? 'max-w-3xl flex h-[80vh] min-h-0 flex-col' : 'max-w-4xl',
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

const emojiTheme = computed<'dark' | 'light'>(() =>
  colorMode.value === 'dark' ? 'dark' : 'light'
)

const emojiGroupNames = computed(() => ({
  smileys_people: t('conversations.compose.emojiGroups.smileysPeople'),
  animals_nature: t('conversations.compose.emojiGroups.animalsNature'),
  food_drink: t('conversations.compose.emojiGroups.foodDrink'),
  activities: t('conversations.compose.emojiGroups.activities'),
  travel_places: t('conversations.compose.emojiGroups.travelPlaces'),
  objects: t('conversations.compose.emojiGroups.objects'),
  symbols: t('conversations.compose.emojiGroups.symbols'),
  flags: t('conversations.compose.emojiGroups.flags'),
  recent: t('conversations.compose.emojiGroups.recent')
}))

function onEmojiSelect(event: EmojiPickerSelectEvent) {
  richEditorRef.value?.insertAtCursor(event.i)
  emojiOpen.value = false
}

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

async function send() {
  const text = reply.value.trim()
  const readyAttachments = attachments.value.filter(a => a.url && !a.error)
  if (!auth.account?.id || (!text && !readyAttachments.length)) return
  if (charExceeded.value) return
  sending.value = true

  const echoId = crypto.randomUUID()
  const isPrivate = mode.value === 'private'
  const now = new Date().toISOString()
  const replyTarget = messages.replyingTo[props.conversation.id] ?? null
  const replyAuthorIsAgent = replyTarget?.sender?.type === 'user' || replyTarget?.senderType === 'USER'
  const inReplyTo = replyTarget
    ? {
        id: replyTarget.id,
        content: replyTarget.content ?? '',
        author: replyAuthorIsAgent
          ? (auth.user?.name ?? '')
          : (props.conversation.meta?.sender?.name ?? '')
      }
    : undefined
  const contentAttrsObj: Record<string, unknown> = { format: 'markdown', echo_id: echoId }
  if (inReplyTo) contentAttrsObj.in_reply_to = inReplyTo
  const optimistic: Message = {
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
    contentAttributes: contentAttrsObj,
    attachments: readyAttachments.map((a, idx) => ({
      id: -(idx + 1),
      messageId: -1,
      fileType: a.file.type || 'file',
      fileKey: a.url,
      contentType: a.file.type,
      size: a.file.size,
      createdAt: now
    })),
    createdAt: now,
    updatedAt: now
  }
  messages.upsert(optimistic)

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
        file_type: a.file.type,
        size: a.file.size
      }))
    }
    const res = await api<Message>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`, {
      method: 'POST',
      body
    })
    if (res?.id) {
      messages.upsert({ ...res, echoId: res.echoId ?? echoId })
    }
    reply.value = ''
    messages.clearReplyTarget(props.conversation.id)
    for (const url of objectUrls.values()) URL.revokeObjectURL(url)
    objectUrls.clear()
    attachments.value = []
    selectedFiles.value = []
  } catch (err) {
    messages.remove(`tmp:${echoId}`)
    throw err
  } finally {
    sending.value = false
  }
}

const replyTarget = computed(() => messages.replyingTo[props.conversation.id] ?? null)
const replyAuthor = computed(() => {
  const r = replyTarget.value
  if (!r) return ''
  const isAgent = r.sender?.type === 'user' || r.senderType === 'USER'
  if (isAgent) return auth.user?.name ?? t('conversations.message.actions.reply')
  return props.conversation.meta?.sender?.name ?? t('conversations.message.actions.reply')
})
function cancelReply() {
  messages.clearReplyTarget(props.conversation.id)
}

function handleSubmit() {
  send()
}

function handleFilesSelected(files: File[] | null | undefined) {
  if (!files || !auth.account?.id) return
  for (const file of files) {
    uploadFile(file)
  }
}

function patchAttachment(id: string, patch: Partial<UploadedFile>) {
  const idx = attachments.value.findIndex(a => a.id === id)
  if (idx >= 0) Object.assign(attachments.value[idx]!, patch)
}

async function uploadFile(file: File) {
  const id = crypto.randomUUID()
  attachments.value.push({ id, file, uploading: true })

  try {
    const accountId = auth.account!.id
    const form = new FormData()
    form.append('file', file, file.name)

    const res = await api<{ path: string }>(`/accounts/${accountId}/uploads`, {
      method: 'POST',
      body: form
    })

    patchAttachment(id, { url: res.path, uploading: false })
  } catch (err) {
    console.error('[ConversationComposer] upload failed', err)
    patchAttachment(id, { uploading: false, error: 'Upload failed' })
  }
}

function removeAttachment(id: string) {
  const att = attachments.value.find(a => a.id === id)
  if (att) {
    selectedFiles.value = selectedFiles.value.filter(f => f !== att.file)
    const url = objectUrls.get(id)
    if (url) {
      URL.revokeObjectURL(url)
      objectUrls.delete(id)
    }
  }
  attachments.value = attachments.value.filter(a => a.id !== id)
}

const objectUrls = new Map<string, string>()

function isAudioFile(file: File): boolean {
  return !!file.type && file.type.toLowerCase().startsWith('audio')
}

function getObjectUrl(id: string): string | undefined {
  const existing = objectUrls.get(id)
  if (existing) return existing
  const att = attachments.value.find(a => a.id === id)
  if (!att) return undefined
  const url = URL.createObjectURL(att.file)
  objectUrls.set(id, url)
  return url
}

const audioAttachments = computed(() => attachments.value.filter(a => isAudioFile(a.file)))
const nonAudioAttachments = computed(() => attachments.value.filter(a => !isAudioFile(a.file)))

onBeforeUnmount(() => {
  for (const url of objectUrls.values()) URL.revokeObjectURL(url)
  objectUrls.clear()
})

function openFilePicker() {
  const comp = fileUploadRef.value as { $el?: HTMLElement } | null
  const el = comp?.$el ?? null
  const input = el?.querySelector?.('input[type="file"]') as HTMLInputElement | null
  input?.click()
}

onMounted(() => {
  if (auth.account?.id && !canned.list.length) {
    api<{ payload: { id: string, shortCode: string, content: string }[] }>(`/accounts/${auth.account.id}/canned_responses`)
      .then((res) => {
        if (res.payload) {
          for (const item of res.payload) {
            canned.upsert({
              id: item.id,
              accountId: auth.account!.id,
              shortCode: item.shortCode,
              content: item.content,
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString()
            })
          }
        }
      })
      .catch(() => { /* ignore */ })
  }
})
</script>

<template>
  <div
    :class="expanded
      ? 'fixed inset-0 z-40 flex flex-col items-center justify-center bg-default/70 px-4 py-6 backdrop-blur'
      : 'shrink-0 border-t border-default/80 bg-default/95 px-2 py-1.5 sm:px-4'"
    @click.self="expanded && closeExpanded()"
  >
    <ClientOnly>
      <ConversationsRichTextComposer
        ref="richEditorRef"
        v-model="reply"
        :placeholder="promptPlaceholder"
        :disabled="sending"
        :ui-class="composerShellClass"
        @submit="handleSubmit"
      >
        <template #header>
          <div class="mb-2 flex items-center justify-between gap-2">
            <div
              class="inline-flex shrink-0 items-center gap-0.5 rounded-md p-0.5 ring transition-colors"
              :class="mode === 'private'
                ? 'bg-warning/10 ring-warning/25'
                : 'bg-default ring-default'"
            >
              <UButton
                :label="t('conversations.compose.replyTab')"
                color="neutral"
                :variant="mode === 'reply' ? 'soft' : 'ghost'"
                size="xs"
                @click="mode = 'reply'"
              />
              <UButton
                :label="t('conversations.compose.privateTab')"
                :color="mode === 'private' ? 'warning' : 'neutral'"
                :variant="mode === 'private' ? 'soft' : 'ghost'"
                size="xs"
                @click="mode = 'private'"
              />
            </div>
            <span
              v-if="maxChars > 0"
              class="truncate text-xs"
              :class="charExceeded ? 'font-medium text-error' : 'text-dimmed'"
            >
              {{ charCount }}/{{ maxChars }}
            </span>
          </div>

          <div
            v-if="replyTarget"
            class="mb-2 flex items-start gap-2 rounded-md border-l-2 border-primary bg-elevated/70 px-2 py-1 ring ring-default"
          >
            <UIcon name="i-lucide-reply" class="mt-0.5 size-3.5 shrink-0 text-primary" />
            <div class="min-w-0 flex-1">
              <div class="text-[11px] font-medium text-primary">
                {{ replyAuthor }}
              </div>
              <div class="line-clamp-2 whitespace-pre-wrap break-words text-xs text-muted">
                {{ replyTarget.content || t('conversations.message.actions.attachment') }}
              </div>
            </div>
            <UButton
              icon="i-lucide-x"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('common.close')"
              @click="cancelReply"
            />
          </div>

          <div v-if="isRecording" class="mb-2">
            <ConversationsAudioRecorder
              @recorded="onRecorded"
              @canceled="onRecorderCanceled"
              @error="onRecorderError"
            />
          </div>

          <div v-if="audioAttachments.length" class="mb-2 flex flex-col gap-1.5">
            <div
              v-for="att in audioAttachments"
              :key="att.id"
              class="flex items-center gap-2 rounded-md bg-elevated/70 px-2 py-1 ring ring-default"
            >
              <UIcon
                v-if="att.uploading"
                name="i-lucide-loader-2"
                class="size-4 shrink-0 animate-spin text-muted"
              />
              <UIcon
                v-else-if="att.error"
                name="i-lucide-alert-circle"
                class="size-4 shrink-0 text-error"
              />
              <ConversationsAudioPlayer
                v-if="getObjectUrl(att.id)"
                :src="getObjectUrl(att.id)!"
                variant="incoming"
                class="flex-1"
              />
              <button
                type="button"
                class="shrink-0 text-muted transition-colors hover:text-error"
                :aria-label="t('conversations.compose.removeAttachment')"
                @click.stop="removeAttachment(att.id)"
              >
                <UIcon name="i-lucide-x" class="size-4" />
              </button>
            </div>
          </div>

          <UFileUpload
            ref="fileUploadRef"
            v-model="selectedFiles"
            multiple
            :dropzone="false"
            :interactive="false"
            :preview="nonAudioAttachments.length > 0"
            :file-delete="false"
            accept="image/*,audio/*,.pdf,.doc,.docx,.txt,.csv"
            :class="nonAudioAttachments.length ? 'w-full' : 'h-0 overflow-hidden opacity-0'"
            @update:model-value="handleFilesSelected"
          >
            <template #file-trailing="{ file }">
              <template v-for="att in attachments" :key="att.id">
                <div v-if="att.file === file" class="flex items-center gap-1">
                  <UIcon
                    v-if="att.uploading"
                    name="i-lucide-loader-2"
                    class="size-3.5 animate-spin text-muted"
                  />
                  <UIcon
                    v-else-if="att.error"
                    name="i-lucide-alert-circle"
                    class="size-3.5 text-error"
                  />
                  <button
                    type="button"
                    class="text-muted transition-colors hover:text-error"
                    @click.stop="removeAttachment(att.id)"
                  >
                    <UIcon name="i-lucide-x" class="size-3.5" />
                  </button>
                </div>
              </template>
            </template>
          </UFileUpload>
        </template>

        <template #footer>
          <div class="flex w-full min-w-0 items-center gap-1.5">
            <div class="flex min-w-0 flex-1 items-center gap-0.5">
              <UPopover :content="{ align: 'end' }">
                <UTooltip :text="t('conversations.compose.format')">
                  <UButton
                    icon="i-lucide-type"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    class="hidden sm:inline-flex"
                    :aria-label="t('conversations.compose.format')"
                  />
                </UTooltip>

                <template #content>
                  <div class="flex items-center gap-0.5 p-1">
                    <UTooltip :text="t('conversations.compose.toolbar.bold')">
                      <UButton
                        icon="i-lucide-bold"
                        color="neutral"
                        :variant="richEditorRef?.isActive('bold') ? 'soft' : 'ghost'"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.bold')"
                        @click="richEditorRef?.toggleBold()"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.italic')">
                      <UButton
                        icon="i-lucide-italic"
                        color="neutral"
                        :variant="richEditorRef?.isActive('italic') ? 'soft' : 'ghost'"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.italic')"
                        @click="richEditorRef?.toggleItalic()"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.link')">
                      <UButton
                        icon="i-lucide-link"
                        color="neutral"
                        :variant="richEditorRef?.isActive('link') ? 'soft' : 'ghost'"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.link')"
                        @click="promptLink"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.undo')">
                      <UButton
                        icon="i-lucide-undo-2"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.undo')"
                        @click="richEditorRef?.undo()"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.redo')">
                      <UButton
                        icon="i-lucide-redo-2"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.redo')"
                        @click="richEditorRef?.redo()"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.list')">
                      <UButton
                        icon="i-lucide-list"
                        color="neutral"
                        :variant="richEditorRef?.isActive('bulletList') ? 'soft' : 'ghost'"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.list')"
                        @click="richEditorRef?.toggleBulletList()"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.orderedList')">
                      <UButton
                        icon="i-lucide-list-ordered"
                        color="neutral"
                        :variant="richEditorRef?.isActive('orderedList') ? 'soft' : 'ghost'"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.orderedList')"
                        @click="richEditorRef?.toggleOrderedList()"
                      />
                    </UTooltip>
                    <UTooltip :text="t('conversations.compose.toolbar.code')">
                      <UButton
                        icon="i-lucide-code-2"
                        color="neutral"
                        :variant="richEditorRef?.isActive('code') ? 'soft' : 'ghost'"
                        size="xs"
                        :aria-label="t('conversations.compose.toolbar.code')"
                        @click="richEditorRef?.toggleCode()"
                      />
                    </UTooltip>
                  </div>
                </template>
              </UPopover>

              <UPopover v-model:open="emojiOpen" :content="{ align: 'end', side: 'top' }">
                <UTooltip :text="t('conversations.compose.emoji')">
                  <UButton
                    icon="i-lucide-smile"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    :aria-label="t('conversations.compose.emoji')"
                  />
                </UTooltip>
                <template #content>
                  <EmojiPicker
                    :theme="emojiTheme"
                    :native="true"
                    :group-names="emojiGroupNames"
                    :hide-group-icons="false"
                    @select="onEmojiSelect"
                  />
                </template>
              </UPopover>
              <UTooltip :text="t('conversations.compose.attach')">
                <UButton
                  icon="i-lucide-paperclip"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :aria-label="t('conversations.compose.attach')"
                  @click="openFilePicker"
                />
              </UTooltip>
              <UTooltip :text="t('conversations.compose.voice')">
                <UButton
                  icon="i-lucide-mic"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :disabled="isRecording"
                  :aria-label="t('conversations.compose.voice')"
                  @click="startRecording"
                />
              </UTooltip>
              <UPopover v-model:open="cannedOpen" :content="{ align: 'end', side: 'top' }">
                <UTooltip :text="t('conversations.compose.canned')">
                  <UButton
                    icon="i-lucide-quote"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    class="hidden md:inline-flex"
                    :aria-label="t('conversations.compose.canned')"
                  />
                </UTooltip>
                <template #content>
                  <ConversationsCannedResponsePicker
                    :search="cannedSearch"
                    @select="handleCannedSelect"
                  />
                </template>
              </UPopover>
              <UTooltip :text="expanded ? t('conversations.compose.collapse') : t('conversations.compose.expand')">
                <UButton
                  :icon="expanded ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2'"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  class="hidden md:inline-flex"
                  :aria-label="expanded ? t('conversations.compose.collapse') : t('conversations.compose.expand')"
                  @click="expanded = !expanded"
                />
              </UTooltip>
            </div>

            <UButton
              :label="t('conversations.compose.send')"
              trailing-icon="i-lucide-corner-down-left"
              :color="mode === 'private' ? 'warning' : 'primary'"
              size="sm"
              class="shrink-0"
              :loading="sending"
              :disabled="charExceeded || (!reply.trim() && !attachments.length)"
              @click="handleSubmit"
            />
          </div>
        </template>
      </ConversationsRichTextComposer>
    </ClientOnly>
  </div>
</template>
