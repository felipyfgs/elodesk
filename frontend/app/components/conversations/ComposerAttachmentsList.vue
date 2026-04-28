<script setup lang="ts">
interface UploadedFile {
  id: string
  file: File
  url?: string
  uploading: boolean
  error?: string
}

const props = defineProps<{
  attachments: UploadedFile[]
}>()

const emit = defineEmits<{
  remove: [id: string]
}>()

const { t } = useI18n()

const objectUrls = new Map<string, string>()

function isAudioFile(file: File): boolean {
  return !!file.type && file.type.toLowerCase().startsWith('audio')
}

function iconForFile(file: File): string {
  const t = (file.type || '').toLowerCase()
  if (t.startsWith('image/')) return 'i-lucide-image'
  if (t.startsWith('video/')) return 'i-lucide-video'
  if (t.startsWith('audio/')) return 'i-lucide-music'
  if (t.includes('pdf')) return 'i-lucide-file-text'
  if (t.includes('zip') || t.includes('rar') || t.includes('7z')) return 'i-lucide-archive'
  if (t.includes('sheet') || t.includes('excel') || t.includes('csv')) return 'i-lucide-file-spreadsheet'
  if (t.includes('word') || t.includes('document')) return 'i-lucide-file-text'
  if (t.includes('presentation') || t.includes('powerpoint')) return 'i-lucide-presentation'
  return 'i-lucide-paperclip'
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function getObjectUrl(id: string): string | undefined {
  const existing = objectUrls.get(id)
  if (existing) return existing
  const att = props.attachments.find(a => a.id === id)
  if (!att) return undefined
  const url = URL.createObjectURL(att.file)
  objectUrls.set(id, url)
  return url
}

function removeAttachment(id: string) {
  const url = objectUrls.get(id)
  if (url) {
    URL.revokeObjectURL(url)
    objectUrls.delete(id)
  }
  emit('remove', id)
}

const audioAttachments = computed(() => props.attachments.filter(a => isAudioFile(a.file)))
const nonAudioAttachments = computed(() => props.attachments.filter(a => !isAudioFile(a.file)))

onBeforeUnmount(() => {
  for (const url of objectUrls.values()) URL.revokeObjectURL(url)
  objectUrls.clear()
})
</script>

<template>
  <div>
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

    <div v-if="nonAudioAttachments.length" class="mb-2 flex flex-wrap gap-1.5">
      <div
        v-for="att in nonAudioAttachments"
        :key="att.id"
        class="flex max-w-full items-center gap-2 rounded-md bg-elevated/70 px-2 py-1 text-xs ring ring-default"
      >
        <UIcon
          v-if="att.uploading"
          name="i-lucide-loader-2"
          class="size-3.5 shrink-0 animate-spin text-muted"
        />
        <UIcon
          v-else-if="att.error"
          name="i-lucide-alert-circle"
          class="size-3.5 shrink-0 text-error"
        />
        <UIcon
          v-else
          :name="iconForFile(att.file)"
          class="size-3.5 shrink-0 text-muted"
        />
        <span class="max-w-[180px] truncate font-medium">{{ att.file.name }}</span>
        <span class="shrink-0 text-dimmed">{{ formatBytes(att.file.size) }}</span>
        <button
          type="button"
          class="shrink-0 text-muted transition-colors hover:text-error"
          :aria-label="t('conversations.compose.removeAttachment')"
          @click.stop="removeAttachment(att.id)"
        >
          <UIcon name="i-lucide-x" class="size-3.5" />
        </button>
      </div>
    </div>
  </div>
</template>
