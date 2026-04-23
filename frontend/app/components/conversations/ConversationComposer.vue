<script setup lang="ts">
import type { Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useCannedResponsesStore } from '~/stores/cannedResponses'

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

const reply = ref('')
const sending = ref(false)
const mode = ref<'reply' | 'private'>('reply')
const attachments = ref<UploadedFile[]>([])
const selectedFiles = ref<File[]>([])
const fileUploadRef = ref<HTMLElement | null>(null)

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

const contextLabel = computed(() => {
  if (mode.value === 'private') return t('conversations.compose.privateContext')
  return props.conversation.inbox?.name || t('conversations.compose.replyContext')
})

const formattingActions = computed(() => [
  { icon: 'i-lucide-bold', label: t('conversations.compose.toolbar.bold') },
  { icon: 'i-lucide-italic', label: t('conversations.compose.toolbar.italic') },
  { icon: 'i-lucide-link', label: t('conversations.compose.toolbar.link') },
  { icon: 'i-lucide-undo-2', label: t('conversations.compose.toolbar.undo') },
  { icon: 'i-lucide-redo-2', label: t('conversations.compose.toolbar.redo') },
  { icon: 'i-lucide-list', label: t('conversations.compose.toolbar.list') },
  { icon: 'i-lucide-list-ordered', label: t('conversations.compose.toolbar.orderedList') },
  { icon: 'i-lucide-code-2', label: t('conversations.compose.toolbar.code') }
])

async function send() {
  const text = reply.value.trim()
  if (!auth.account?.id || (!text && !attachments.value.length)) return
  if (charExceeded.value) return
  sending.value = true
  try {
    const body: Record<string, unknown> = {
      message: text,
      private: mode.value === 'private'
    }
    if (attachments.value.length) {
      body.attachments = attachments.value
        .filter(a => a.url)
        .map(a => ({ url: a.url, type: a.file.type }))
    }
    await api(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`, {
      method: 'POST',
      body
    })
    reply.value = ''
    attachments.value = []
    selectedFiles.value = []
  } finally {
    sending.value = false
  }
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

async function uploadFile(file: File) {
  const id = crypto.randomUUID()
  const att: UploadedFile = { id, file, uploading: true }
  attachments.value.push(att)

  try {
    const accountId = auth.account!.id
    const res = await api<{ url: string, key: string }>(`/accounts/${accountId}/uploads/signed-url`, {
      method: 'POST',
      body: {
        file_name: file.name,
        file_type: file.type,
        file_size: file.size
      }
    })

    await $fetch(res.url, {
      method: 'PUT',
      body: file,
      headers: { 'Content-Type': file.type }
    })

    att.url = res.key
    att.uploading = false
  } catch {
    att.uploading = false
    att.error = 'Upload failed'
  }
}

function removeAttachment(id: string) {
  const att = attachments.value.find(a => a.id === id)
  if (att) {
    selectedFiles.value = selectedFiles.value.filter(f => f !== att.file)
  }
  attachments.value = attachments.value.filter(a => a.id !== id)
}

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
  <div class="shrink-0 border-t border-default/80 bg-default/95 px-2 py-2 sm:px-4">
    <UChatPrompt
      v-model="reply"
      :placeholder="promptPlaceholder"
      :rows="1"
      :autoresize="true"
      :maxrows="6"
      :disabled="sending"
      variant="outline"
      class="mx-auto max-w-3xl"
      :ui="{
        root: 'gap-3 rounded-lg bg-elevated/85 px-3 py-2 shadow-lg ring ring-default',
        header: 'w-full',
        body: 'min-h-12',
        base: 'min-h-12 text-sm leading-5',
        footer: 'w-full'
      }"
      @submit="handleSubmit"
    >
      <template #header>
        <div class="flex w-full flex-col gap-2">
          <UFileUpload
            ref="fileUploadRef"
            v-model="selectedFiles"
            multiple
            :dropzone="false"
            :interactive="false"
            :preview="attachments.length > 0"
            :file-delete="false"
            accept="image/*,.pdf,.doc,.docx,.txt,.csv"
            :class="attachments.length ? 'w-full' : 'h-0 overflow-hidden opacity-0'"
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

          <div class="flex items-center justify-between gap-2">
            <div class="inline-flex min-w-0 items-center gap-1 rounded-lg bg-default p-0.5 ring ring-default">
              <UButton
                :label="t('conversations.compose.replyTab')"
                color="neutral"
                :variant="mode === 'reply' ? 'soft' : 'ghost'"
                size="xs"
                class="min-w-0"
                @click="mode = 'reply'"
              />
              <UButton
                :label="t('conversations.compose.privateTab')"
                color="neutral"
                :variant="mode === 'private' ? 'soft' : 'ghost'"
                size="xs"
                class="min-w-0"
                @click="mode = 'private'"
              />
            </div>

            <UTooltip :text="t('conversations.compose.expand')">
              <UButton
                icon="i-lucide-maximize-2"
                color="neutral"
                variant="ghost"
                size="xs"
                disabled
                :aria-label="t('conversations.compose.expand')"
              />
            </UTooltip>
          </div>
        </div>
      </template>

      <template #footer>
        <div class="flex w-full flex-col gap-3">
          <div class="flex items-center gap-1 overflow-x-auto">
            <UTooltip
              v-for="action in formattingActions"
              :key="action.icon"
              :text="action.label"
            >
              <UButton
                :icon="action.icon"
                color="neutral"
                variant="ghost"
                size="xs"
                disabled
                :aria-label="action.label"
              />
            </UTooltip>
          </div>

          <div class="flex items-end justify-between gap-3">
            <div class="flex min-w-0 items-center gap-1.5 text-xs text-muted">
              <UIcon
                :name="mode === 'private' ? 'i-lucide-lock' : 'i-lucide-message-square'"
                class="size-3.5 shrink-0"
              />
              <span class="truncate">{{ contextLabel }}</span>
              <span
                v-if="maxChars > 0"
                class="shrink-0"
                :class="charExceeded ? 'font-medium text-error' : 'text-dimmed'"
              >
                {{ charCount }}/{{ maxChars }}
              </span>
            </div>

            <div class="flex shrink-0 items-center gap-1.5">
              <UTooltip :text="t('conversations.compose.emoji')">
                <UButton
                  icon="i-lucide-smile"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  disabled
                  :aria-label="t('conversations.compose.emoji')"
                />
              </UTooltip>
              <UTooltip :text="t('conversations.compose.attach')">
                <UButton
                  icon="i-lucide-paperclip"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  :aria-label="t('conversations.compose.attach')"
                  @click="openFilePicker"
                />
              </UTooltip>
              <UTooltip :text="t('conversations.compose.voice')">
                <UButton
                  icon="i-lucide-mic"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  disabled
                  :aria-label="t('conversations.compose.voice')"
                />
              </UTooltip>
              <UTooltip :text="t('conversations.compose.canned')">
                <UButton
                  icon="i-lucide-quote"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  disabled
                  :aria-label="t('conversations.compose.canned')"
                />
              </UTooltip>
              <UButton
                type="submit"
                :label="t('conversations.compose.send')"
                trailing-icon="i-lucide-corner-down-left"
                color="primary"
                size="sm"
                :loading="sending"
                :disabled="charExceeded || (!reply.trim() && !attachments.length)"
              />
            </div>
          </div>
        </div>
      </template>
    </UChatPrompt>
  </div>
</template>
