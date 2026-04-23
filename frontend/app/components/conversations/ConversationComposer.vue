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
const attachments = ref<UploadedFile[]>([])
const selectedFiles = ref<File[]>([])
const fileUploadRef = ref<HTMLElement | null>(null)

const chatStatus = computed<'ready' | 'submitted' | 'error'>(() => {
  if (sending.value) return 'submitted'
  return 'ready'
})

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

async function send() {
  if (!auth.account?.id || (!reply.value.trim() && !attachments.value.length)) return
  if (charExceeded.value) return
  sending.value = true
  try {
    const body: Record<string, unknown> = {
      content: reply.value.trim() || null
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
  const el = fileUploadRef.value?.$el ?? fileUploadRef.value
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
  <div class="pb-4 px-4 sm:px-6 shrink-0">
    <UChatPrompt
      v-model="reply"
      :placeholder="t('conversations.compose.placeholder')"
      :rows="3"
      :disabled="sending"
      variant="outline"
      @submit="handleSubmit"
    >
      <template #header>
        <UFileUpload
          ref="fileUploadRef"
          v-model="selectedFiles"
          multiple
          :dropzone="false"
          :interactive="false"
          :preview="attachments.length > 0"
          :file-delete="false"
          accept="image/*,.pdf,.doc,.docx,.txt,.csv"
          class="w-full"
          @update:model-value="handleFilesSelected"
        >
          <template #file-trailing="{ file }">
            <template v-for="att in attachments" :key="att.id">
              <div v-if="att.file === file" class="flex items-center gap-1">
                <UIcon
                  v-if="att.uploading"
                  name="i-lucide-loader-2"
                  class="size-3.5 text-muted animate-spin"
                />
                <UIcon
                  v-else-if="att.error"
                  name="i-lucide-alert-circle"
                  class="size-3.5 text-error"
                />
                <button
                  type="button"
                  class="text-muted hover:text-error transition-colors"
                  @click.stop="removeAttachment(att.id)"
                >
                  <UIcon name="i-lucide-x" class="size-3.5" />
                </button>
              </div>
            </template>
          </template>
        </UFileUpload>
      </template>

      <template #leading>
        <UTooltip :text="t('conversations.compose.attach')">
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-lucide-paperclip"
            size="xs"
            @click="openFilePicker"
          />
        </UTooltip>

        <span
          v-if="maxChars > 0"
          class="text-xs"
          :class="charExceeded ? 'text-error font-medium' : 'text-dimmed'"
        >
          {{ charCount }}/{{ maxChars }}
        </span>
      </template>

      <template #trailing>
        <UChatPromptSubmit
          :status="chatStatus"
          :disabled="charExceeded || (!reply.trim() && !attachments.length)"
        />
      </template>
    </UChatPrompt>
  </div>
</template>
