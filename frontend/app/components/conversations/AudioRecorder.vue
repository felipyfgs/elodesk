<script setup lang="ts">
import WaveSurfer from 'wavesurfer.js'
import RecordPlugin from 'wavesurfer.js/dist/plugins/record.esm.js'

const emit = defineEmits<{
  recorded: [file: File]
  canceled: []
  error: [reason: 'permissionDenied' | 'unavailable' | 'unsupported']
}>()

const { t } = useI18n()

const containerRef = ref<HTMLDivElement | null>(null)
const duration = ref(0)
const state = ref<'idle' | 'recording' | 'paused'>('idle')

let ws: WaveSurfer | null = null
let record: ReturnType<typeof RecordPlugin.create> | null = null
let recordedMime = ''

// Bitrate "PTT do WhatsApp": 32 kbps em mono é o que o app oficial usa para
// voice notes. Bate o ouvido humano de voz sem inflar o arquivo.
const PTT_BITRATE = 32000

// Padrão de mercado para voice note: OGG/Opus. Firefox e Chromium 113+
// gravam ogg/opus diretamente via MediaRecorder; Safari ainda não, então
// caímos em webm/opus e o wzap re-encoda com ffmpeg antes de despachar pro
// WhatsApp. A prioridade aqui garante .ogg sempre que o browser permitir.
const PREFERRED_MIME_TYPES = [
  'audio/ogg;codecs=opus',
  'audio/ogg',
  'audio/webm;codecs=opus',
  'audio/webm'
]

function pickAudioMimeType(): string {
  if (typeof MediaRecorder === 'undefined') return ''
  for (const mt of PREFERRED_MIME_TYPES) {
    if (MediaRecorder.isTypeSupported(mt)) return mt
  }
  return ''
}

function extensionForMime(mime: string): string {
  const m = mime.toLowerCase()
  if (m.includes('ogg')) return 'ogg'
  if (m.includes('mp4') || m.includes('m4a') || m.includes('aac')) return 'm4a'
  if (m.includes('wav')) return 'wav'
  if (m.includes('mpeg') || m.includes('mp3')) return 'mp3'
  return 'webm'
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = Math.floor(seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

async function start() {
  if (!containerRef.value) return
  if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
    emit('error', 'unsupported')
    return
  }

  const primary = cssVar('--ui-primary', '#22c55e')

  ws = WaveSurfer.create({
    container: containerRef.value,
    waveColor: primary,
    progressColor: primary,
    cursorWidth: 0,
    barWidth: 2,
    barGap: 2,
    barRadius: 2,
    height: 32,
    interact: false
  })

  recordedMime = pickAudioMimeType()
  const recordOpts: Record<string, unknown> = {
    scrollingWaveform: true,
    renderRecordedAudio: false,
    audioBitsPerSecond: PTT_BITRATE
  }
  // Só injeta `mimeType` quando algum candidato passou no isTypeSupported —
  // o RecordPlugin valida internamente e quebra se a string não for suportada
  // (ex.: passar 'audio/ogg' no Safari faz a gravação nem iniciar).
  if (recordedMime) recordOpts.mimeType = recordedMime
  record = ws.registerPlugin(RecordPlugin.create(recordOpts))

  record.on('record-progress', (time: number) => {
    duration.value = time / 1000
  })

  try {
    await record.startRecording()
    state.value = 'recording'
  } catch (err) {
    const name = (err as DOMException | undefined)?.name
    if (name === 'NotAllowedError' || name === 'SecurityError') emit('error', 'permissionDenied')
    else if (name === 'NotFoundError' || name === 'NotReadableError') emit('error', 'unavailable')
    else emit('error', 'unsupported')
    destroy()
  }
}

function togglePause() {
  if (!record) return
  if (state.value === 'recording') {
    record.pauseRecording()
    state.value = 'paused'
  } else if (state.value === 'paused') {
    record.resumeRecording()
    state.value = 'recording'
  }
}

async function stop() {
  if (!record) return
  const blob: Blob = await new Promise((resolve) => {
    const off = record!.on('record-end', (b: Blob) => {
      off()
      resolve(b)
    })
    record!.stopRecording()
  })
  // `recordedMime` é o que o MediaRecorder aceitou; `blob.type` às vezes vem
  // vazio (Firefox antigo) ou normalizado (sem ;codecs=opus). Preferimos o
  // mime que pedimos, pra preservar a indicação de codec até o wzap.
  const mime = recordedMime || blob.type || 'audio/webm'
  const ext = extensionForMime(mime)
  const file = new File([blob], `voice-${Date.now()}.${ext}`, { type: mime })
  destroy()
  emit('recorded', file)
}

function cancel() {
  destroy()
  emit('canceled')
}

function destroy() {
  record?.stopRecording()
  ws?.destroy()
  ws = null
  record = null
  recordedMime = ''
  duration.value = 0
  state.value = 'idle'
}

onMounted(() => {
  nextTick(() => start())
})

onBeforeUnmount(() => {
  destroy()
})
</script>

<template>
  <div class="flex w-full items-center gap-2 rounded-md bg-error/5 px-2 py-1.5 ring ring-error/20">
    <UTooltip :text="t('conversations.compose.voiceCancel')">
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('conversations.compose.voiceCancel')"
        @click="cancel"
      />
    </UTooltip>

    <span class="relative flex size-2 shrink-0">
      <span
        class="absolute inline-flex h-full w-full rounded-full bg-error opacity-75"
        :class="state === 'recording' ? 'animate-ping' : ''"
      />
      <span class="relative inline-flex size-2 rounded-full bg-error" />
    </span>

    <span class="shrink-0 font-mono text-xs tabular-nums text-muted">
      {{ formatDuration(duration) }}
    </span>

    <div ref="containerRef" class="min-w-0 flex-1" />

    <UTooltip :text="state === 'paused' ? t('conversations.compose.voiceResume') : t('conversations.compose.voicePause')">
      <UButton
        :icon="state === 'paused' ? 'i-lucide-play' : 'i-lucide-pause'"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="state === 'paused' ? t('conversations.compose.voiceResume') : t('conversations.compose.voicePause')"
        @click="togglePause"
      />
    </UTooltip>

    <UTooltip :text="t('conversations.compose.voiceStop')">
      <UButton
        icon="i-lucide-check"
        color="primary"
        variant="solid"
        size="xs"
        :aria-label="t('conversations.compose.voiceStop')"
        @click="stop"
      />
    </UTooltip>
  </div>
</template>
