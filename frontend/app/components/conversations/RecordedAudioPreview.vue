<script setup lang="ts">
import WaveSurfer from 'wavesurfer.js'

interface UploadedFile {
  id: string
  file: File
  url?: string
  uploading: boolean
  error?: string
  isRecordedAudio?: boolean
}

const props = defineProps<{
  attachment: UploadedFile
}>()

const emit = defineEmits<{
  remove: [id: string]
}>()

const { t } = useI18n()

const containerRef = ref<HTMLDivElement | null>(null)
const isPlaying = ref(false)
const currentTime = ref(0)
const totalDuration = ref(0)

let ws: WaveSurfer | null = null
let objectUrl: string | null = null

function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '00:00'
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = Math.floor(seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function init() {
  if (!containerRef.value) return
  const primary = cssVar('--ui-primary', '#22c55e')
  const muted = cssVar('--ui-text-muted', '#9ca3af')

  // ObjectURL local — o blob do recorder vive em memória até o destroy.
  // O upload pra S3/MinIO acontece em paralelo (uploadFile no Composer);
  // aqui só queremos um buffer pra reproduzir antes do envio.
  objectUrl = URL.createObjectURL(props.attachment.file)

  ws = WaveSurfer.create({
    container: containerRef.value,
    waveColor: muted,
    progressColor: primary,
    cursorWidth: 1,
    barWidth: 2,
    barGap: 2,
    barRadius: 2,
    height: 32,
    interact: true
  })

  ws.on('ready', () => {
    totalDuration.value = ws?.getDuration() ?? 0
  })
  ws.on('audioprocess', () => {
    currentTime.value = ws?.getCurrentTime() ?? 0
  })
  ws.on('seeking', () => {
    currentTime.value = ws?.getCurrentTime() ?? 0
  })
  ws.on('finish', () => {
    isPlaying.value = false
    currentTime.value = 0
  })
  ws.on('play', () => {
    isPlaying.value = true
  })
  ws.on('pause', () => {
    isPlaying.value = false
  })

  void ws.load(objectUrl)
}

function togglePlay() {
  ws?.playPause()
}

function remove() {
  emit('remove', props.attachment.id)
}

onMounted(() => {
  nextTick(() => init())
})

onBeforeUnmount(() => {
  ws?.destroy()
  ws = null
  if (objectUrl) {
    URL.revokeObjectURL(objectUrl)
    objectUrl = null
  }
})
</script>

<template>
  <div class="flex w-full items-center gap-2 rounded-md bg-elevated/70 px-2 py-1.5 ring ring-default">
    <UTooltip :text="isPlaying ? t('conversations.compose.audioPause') : t('conversations.compose.audioPlay')">
      <UButton
        :icon="isPlaying ? 'i-lucide-pause' : 'i-lucide-play'"
        color="primary"
        variant="solid"
        size="xs"
        :aria-label="isPlaying ? t('conversations.compose.audioPause') : t('conversations.compose.audioPlay')"
        @click="togglePlay"
      />
    </UTooltip>

    <span class="shrink-0 font-mono text-xs tabular-nums text-muted">
      {{ formatTime(isPlaying || currentTime > 0 ? currentTime : totalDuration) }}
    </span>

    <div ref="containerRef" class="min-w-0 flex-1" />

    <span v-if="attachment.uploading" class="shrink-0 text-xs text-muted">
      <UIcon name="i-lucide-loader-2" class="size-3.5 animate-spin" />
    </span>
    <span v-else-if="attachment.error" class="shrink-0 text-xs text-error">
      <UIcon name="i-lucide-alert-circle" class="size-3.5" />
    </span>

    <UTooltip :text="t('conversations.compose.removeAttachment')">
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('conversations.compose.removeAttachment')"
        @click="remove"
      />
    </UTooltip>
  </div>
</template>
