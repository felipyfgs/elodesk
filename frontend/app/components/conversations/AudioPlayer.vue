<script setup lang="ts">
import { useAVWaveform } from 'vue-audio-visual'
import { useAuthStore } from '~/stores/auth'
import { useAudioPlayerStore } from '~/stores/audioPlayer'

const props = defineProps<{
  path?: string
  src?: string
  variant?: 'incoming' | 'outgoing'
  trackId?: string
  conversationId?: string | number
  accountId?: string | number
  title?: string
}>()

const { t } = useI18n()
const auth = useAuthStore()
const audioStore = useAudioPlayerStore()
const runtime = useRuntimeConfig()

const audioRef = ref<HTMLAudioElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const resolvedUrl = ref<string | null>(null)
const localDuration = ref(0)
const errored = ref(false)

const isOutgoing = computed(() => props.variant === 'outgoing')

// Per-player identity used to distinguish which AudioPlayer currently
// holds the active track inside the global singleton store.
const uid = computed(() => props.trackId ?? `audio:${props.path ?? props.src ?? ''}`)
const isActive = computed(() => audioStore.track?.id === uid.value)
const playing = computed(() => isActive.value && audioStore.isPlaying)
const displayTime = computed(() => isActive.value ? audioStore.currentTime : 0)
const displayDuration = computed(() => isActive.value && audioStore.duration > 0 ? audioStore.duration : localDuration.value)
const displayRate = computed(() => isActive.value ? audioStore.playbackRate : 1)
const displayMuted = computed(() => isActive.value && audioStore.muted)

let blobUrl: string | null = null
let initialized = false

function format(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '00:00'
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = Math.floor(seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

function initWaveform() {
  if (!resolvedUrl.value || !audioRef.value || !canvasRef.value || initialized) return
  initialized = true
  const primary = cssVar('--ui-primary', '#22c55e')
  useAVWaveform(audioRef, canvasRef, {
    src: resolvedUrl.value,
    canvWidth: 384,
    canvHeight: 64,
    playedLineWidth: 2,
    playedLineColor: isOutgoing.value ? '#ffffff' : primary,
    noplayedLineWidth: 2,
    noplayedLineColor: isOutgoing.value ? 'rgba(255,255,255,0.7)' : 'rgba(148,163,184,0.9)',
    playtime: false,
    playtimeSlider: true,
    playtimeSliderColor: isOutgoing.value ? 'rgba(255,255,255,0.95)' : primary,
    playtimeSliderWidth: 2.5,
    playtimeClickable: true
  })
}

async function loadUrl() {
  if (props.src) {
    resolvedUrl.value = props.src
    return
  }
  if (!props.path || !auth.account?.id) return
  try {
    const apiBase = runtime.public.apiUrl as string
    const url = `${apiBase}/accounts/${auth.account.id}/uploads/download?path=${encodeURIComponent(props.path)}`
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${auth.accessToken}` }
    })
    if (!res.ok) throw new Error(`download failed: ${res.status}`)
    const blob = await res.blob()
    blobUrl = URL.createObjectURL(blob)
    resolvedUrl.value = blobUrl
  } catch (err) {
    console.error('[AudioPlayer] failed to fetch audio', err)
    errored.value = true
  }
}

function togglePlay() {
  if (errored.value) return
  if (isActive.value) {
    audioStore.toggle()
    return
  }
  // Prefer `path` so the store fetches its own authenticated blob — that
  // keeps playback working after this inline player unmounts (e.g. when
  // the user switches conversations). Fall back to `src` for composer
  // previews where no MinIO download is needed.
  audioStore.play({
    id: uid.value,
    path: props.path,
    src: props.path ? undefined : (props.src ?? resolvedUrl.value ?? undefined),
    title: props.title,
    accountId: props.accountId,
    conversationId: props.conversationId
  })
}

function cycleSpeed() {
  const speeds = [1, 1.5, 2]
  const idx = speeds.indexOf(displayRate.value)
  const next = speeds[(idx + 1) % speeds.length] ?? 1
  if (isActive.value) audioStore.setPlaybackRate(next)
}

function toggleMute() {
  if (!isActive.value) return
  audioStore.setMuted(!audioStore.muted)
}

function seekFromCanvas(event: MouseEvent) {
  if (!isActive.value || !audioStore.duration) return
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width))
  audioStore.seek(ratio * audioStore.duration)
}

function download() {
  if (!resolvedUrl.value) return
  const a = document.createElement('a')
  a.href = resolvedUrl.value
  a.download = props.path?.split('/').pop() ?? 'audio'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

function cleanup() {
  if (blobUrl) {
    URL.revokeObjectURL(blobUrl)
    blobUrl = null
  }
  initialized = false
  localDuration.value = 0
  errored.value = false
}

onMounted(async () => {
  await loadUrl()
  await nextTick()
  initWaveform()
})

onBeforeUnmount(cleanup)

watch(() => [props.src, props.path], async () => {
  cleanup()
  await loadUrl()
  await nextTick()
  initWaveform()
})

// Drive the local (silent) audio element's currentTime from the store so
// useAVWaveform's playhead slider tracks the global playback. Reset to 0
// whenever the active track changes away from us.
watch([displayTime, isActive], ([time, active]) => {
  const el = audioRef.value
  if (!el) return
  if (active) {
    const diff = Math.abs(el.currentTime - time)
    if (diff > 0.1) el.currentTime = time
  } else if (el.currentTime !== 0) {
    el.currentTime = 0
  }
})
</script>

<template>
  <div class="flex w-fit max-w-full items-center gap-2">
    <UButton
      :icon="playing ? 'i-lucide-pause' : 'i-lucide-play'"
      :color="isOutgoing ? 'neutral' : 'primary'"
      :variant="isOutgoing ? 'subtle' : 'soft'"
      size="sm"
      class="shrink-0"
      :disabled="!resolvedUrl || errored"
      :aria-label="playing ? t('conversations.audio.pause') : t('conversations.audio.play')"
      @click="togglePlay"
    />

    <audio
      v-if="resolvedUrl"
      ref="audioRef"
      class="hidden"
      :src="resolvedUrl"
      preload="metadata"
      muted
      @loadedmetadata="localDuration = audioRef?.duration ?? 0"
      @error="errored = true"
    />

    <div
      class="audio-waveform h-8 w-40 shrink-0 sm:w-48"
      @click="seekFromCanvas"
    >
      <canvas ref="canvasRef" class="block h-8 w-full" />
    </div>

    <span
      class="shrink-0 font-mono text-xs leading-none tabular-nums"
      :class="isOutgoing ? 'text-white/80' : 'text-muted'"
    >
      {{ errored ? t('conversations.audio.error') : format(isActive ? displayTime : displayDuration) }}
    </span>

    <button
      v-if="!errored && isActive"
      type="button"
      class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium leading-none tabular-nums transition-colors"
      :class="isOutgoing ? 'bg-white/15 text-white hover:bg-white/25' : 'bg-default text-muted ring ring-default hover:bg-elevated'"
      :aria-label="t('conversations.audio.speed')"
      @click="cycleSpeed"
    >
      {{ displayRate }}x
    </button>

    <button
      v-if="!errored && isActive"
      type="button"
      class="shrink-0 grid size-6 place-content-center rounded transition-colors"
      :class="isOutgoing ? 'text-white/80 hover:bg-white/15' : 'text-muted hover:bg-elevated'"
      :aria-label="displayMuted ? t('conversations.audio.unmute') : t('conversations.audio.mute')"
      @click="toggleMute"
    >
      <UIcon :name="displayMuted ? 'i-lucide-volume-off' : 'i-lucide-volume-2'" class="size-3.5" />
    </button>

    <button
      v-if="!errored && resolvedUrl"
      type="button"
      class="shrink-0 grid size-6 place-content-center rounded transition-colors"
      :class="isOutgoing ? 'text-white/80 hover:bg-white/15' : 'text-muted hover:bg-elevated'"
      :aria-label="t('conversations.audio.download')"
      @click="download"
    >
      <UIcon name="i-lucide-download" class="size-3.5" />
    </button>
  </div>
</template>

<style scoped>
.audio-waveform {
  overflow: hidden;
  cursor: pointer;
}

.audio-waveform :deep(> div) {
  display: block;
  width: 100%;
  overflow: hidden;
}
</style>
