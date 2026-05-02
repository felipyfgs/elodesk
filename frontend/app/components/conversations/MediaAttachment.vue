<script setup lang="ts">
import type { MessageAttachment } from '~/utils/chatAdapter'

const props = defineProps<{
  attachment: MessageAttachment
  accountId?: string | number
  conversationId?: string | number
  isSticker?: boolean
}>()

const { src, errored, isLoading } = useAttachmentSrc(props.attachment, props.accountId)
const lightboxOpen = ref(false)

const kind = computed<'image' | 'video' | 'audio' | 'sticker' | 'pdf' | 'file'>(() => {
  if (props.isSticker) return 'sticker'
  const t = (props.attachment.fileType ?? '').toLowerCase()
  const ext = (props.attachment.extension ?? '').toLowerCase().replace(/^\./, '')
  if (t.startsWith('image') || t === 'image') return 'image'
  if (t.startsWith('video') || t === 'video') return 'video'
  if (t.startsWith('audio') || t === 'audio') return 'audio'
  if (t.includes('pdf') || ext === 'pdf') return 'pdf'
  return 'file'
})

const UUID_PREFIX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}-/i

const fileName = computed(() => {
  if (props.attachment.fileName) return props.attachment.fileName
  const path = props.attachment.path
  if (path) {
    const segment = path.split('/').pop() ?? path
    const stripped = segment.replace(UUID_PREFIX, '')
    if (stripped) return stripped
  }
  const ext = (props.attachment.extension ?? '').replace(/^\./, '')
  return ext ? `arquivo.${ext}` : 'arquivo'
})

const fileIcon = computed(() => {
  const ext = (props.attachment.extension ?? '').toLowerCase().replace(/^\./, '')
  const mime = (props.attachment.fileType ?? '').toLowerCase()
  if (ext === 'pdf' || mime.includes('pdf')) return 'i-lucide-file-text'
  if (['xlsx', 'xls', 'csv', 'ods'].includes(ext) || mime.includes('sheet') || mime.includes('csv') || mime.includes('excel')) return 'i-lucide-file-spreadsheet'
  if (['docx', 'doc', 'odt', 'rtf', 'txt'].includes(ext) || mime.includes('word') || mime.includes('document') || mime.includes('text')) return 'i-lucide-file-text'
  if (['pptx', 'ppt', 'odp'].includes(ext) || mime.includes('presentation') || mime.includes('powerpoint')) return 'i-lucide-presentation'
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext) || mime.includes('zip') || mime.includes('archive') || mime.includes('compressed')) return 'i-lucide-archive'
  return 'i-lucide-paperclip'
})

const fileSizeLabel = computed(() => {
  const bytes = props.attachment.size
  if (!bytes) return null
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
})
</script>

<template>
  <div v-if="kind === 'sticker'" class="max-w-[160px]">
    <img
      v-if="src"
      :src="src"
      :alt="$t('conversations.message.sticker', 'Sticker')"
      class="size-auto max-h-[160px] max-w-full object-contain"
      loading="lazy"
    >
  </div>
  <template v-else-if="kind === 'image'">
    <button
      type="button"
      class="block overflow-hidden rounded-md bg-black/10 transition hover:opacity-90"
      :aria-label="$t('common.open', 'Abrir')"
      @click="lightboxOpen = true"
    >
      <div v-if="isLoading" class="flex h-32 w-48 items-center justify-center text-xs text-muted">
        <UIcon name="i-lucide-loader-2" class="size-4 animate-spin" />
      </div>
      <div v-else-if="errored || !src" class="flex h-32 w-48 items-center justify-center text-xs text-muted">
        <UIcon name="i-lucide-image-off" class="size-4" />
      </div>
      <img
        v-else
        :src="src"
        :alt="$t('conversations.message.image', 'Imagem')"
        class="block max-h-[320px] max-w-[320px] object-cover"
        loading="lazy"
      >
    </button>

    <UModal v-model:open="lightboxOpen" :ui="{ content: 'sm:max-w-4xl' }">
      <template #content>
        <div class="flex items-center justify-center bg-black/95 p-4">
          <img
            v-if="src"
            :src="src"
            :alt="$t('conversations.message.image', 'Imagem')"
            class="max-h-[85vh] max-w-full object-contain"
          >
        </div>
      </template>
    </UModal>
  </template>
  <video
    v-else-if="kind === 'video' && src"
    :src="src"
    controls
    preload="metadata"
    class="block max-h-[320px] max-w-[320px] rounded-md bg-black"
  />
  <ConversationsAudioPlayer
    v-else-if="kind === 'audio'"
    :src="src ?? undefined"
    :account-id="accountId"
    :conversation-id="conversationId"
    :track-id="attachment.id ? `att:${attachment.id}` : undefined"
  />
  <ConversationsPdfPreview
    v-else-if="kind === 'pdf' && src"
    :src="src"
    :file-name="fileName"
    :file-size-label="fileSizeLabel"
  />
  <a
    v-else-if="src"
    :href="src"
    target="_blank"
    rel="noopener noreferrer"
    :title="fileName"
    class="flex max-w-[280px] items-center gap-2.5 rounded-md bg-default px-2.5 py-2 text-xs ring ring-default transition-colors hover:bg-elevated"
  >
    <UIcon :name="fileIcon" class="size-5 shrink-0 text-primary" />
    <div class="min-w-0 flex-1">
      <p class="truncate font-medium text-default">
        {{ fileName }}
      </p>
      <p v-if="fileSizeLabel" class="text-dimmed">
        {{ fileSizeLabel }}
      </p>
    </div>
    <UIcon name="i-lucide-download" class="size-4 shrink-0 text-muted" />
  </a>

  <span
    v-else
    :title="fileName"
    class="flex max-w-[280px] items-center gap-2 rounded-md bg-default px-2.5 py-2 text-xs text-muted ring ring-default"
  >
    <UIcon :name="fileIcon" class="size-5 shrink-0" />
    <span class="truncate">{{ fileName }}</span>
  </span>
</template>
