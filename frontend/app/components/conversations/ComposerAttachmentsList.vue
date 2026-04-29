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

function isImageFile(file: File): boolean {
  return !!file.type && file.type.toLowerCase().startsWith('image')
}

function iconForFile(file: File): string {
  const t = (file.type || '').toLowerCase()
  if (t.startsWith('video/')) return 'i-lucide-video'
  if (t.startsWith('audio/')) return 'i-lucide-music'
  if (t.includes('pdf')) return 'i-lucide-file-text'
  if (t.includes('zip') || t.includes('rar') || t.includes('7z')) return 'i-lucide-archive'
  if (t.includes('sheet') || t.includes('excel') || t.includes('csv')) return 'i-lucide-file-spreadsheet'
  if (t.includes('word') || t.includes('document')) return 'i-lucide-file-text'
  if (t.includes('presentation') || t.includes('powerpoint')) return 'i-lucide-presentation'
  return 'i-lucide-paperclip'
}

function iconColorClass(file: File): string {
  const t = (file.type || '').toLowerCase()
  if (t.includes('pdf')) return 'text-error'
  if (t.includes('sheet') || t.includes('excel') || t.includes('csv')) return 'text-success'
  if (t.includes('word') || t.includes('document') || t.includes('text')) return 'text-info'
  if (t.includes('presentation') || t.includes('powerpoint')) return 'text-warning'
  if (t.startsWith('video/')) return 'text-primary'
  if (t.includes('zip') || t.includes('rar') || t.includes('7z')) return 'text-warning'
  return 'text-primary'
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

onBeforeUnmount(() => {
  for (const url of objectUrls.values()) URL.revokeObjectURL(url)
  objectUrls.clear()
})
</script>

<template>
  <div v-if="attachments.length" class="mb-2 flex max-h-40 flex-col gap-1 overflow-y-auto pr-1">
    <div
      v-for="att in attachments"
      :key="att.id"
      class="group flex w-fit max-w-xs items-center gap-2 rounded-md bg-elevated/70 py-1 pl-1 pr-2 text-xs ring ring-default transition-colors hover:bg-elevated"
    >
      <div class="relative size-6 shrink-0 overflow-hidden rounded bg-default">
        <template v-if="isImageFile(att.file) && getObjectUrl(att.id) && !att.error">
          <img
            :src="getObjectUrl(att.id)!"
            :alt="att.file.name"
            class="size-full object-cover"
          >
        </template>
        <div v-else class="grid size-full place-content-center">
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
          <UIcon
            v-else
            :name="iconForFile(att.file)"
            class="size-3.5"
            :class="iconColorClass(att.file)"
          />
        </div>
        <div
          v-if="isImageFile(att.file) && att.uploading"
          class="absolute inset-0 grid place-content-center bg-default/60"
        >
          <UIcon name="i-lucide-loader-2" class="size-3 animate-spin text-default" />
        </div>
      </div>
      <span class="min-w-0 flex-1 truncate font-medium" :title="att.file.name">{{ att.file.name }}</span>
      <span class="shrink-0 tabular-nums text-dimmed">{{ formatBytes(att.file.size) }}</span>
      <button
        type="button"
        class="grid size-5 shrink-0 place-content-center rounded-full text-muted transition-colors hover:bg-error/10 hover:text-error"
        :aria-label="t('conversations.compose.removeAttachment')"
        @click.stop="removeAttachment(att.id)"
      >
        <UIcon name="i-lucide-x" class="size-3" />
      </button>
    </div>
  </div>
</template>
