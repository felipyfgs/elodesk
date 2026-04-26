<script setup lang="ts">
import { useAudioPlayerStore } from '~/stores/audioPlayer'
import { useConversationsStore } from '~/stores/conversations'

const { t } = useI18n()
const audioStore = useAudioPlayerStore()
const conversationsStore = useConversationsStore()

// Hide when user is already viewing the conversation that owns the track —
// the inline AudioPlayer is the canonical UI in that context.
const onSourceConversation = computed(() => {
  const trackConv = audioStore.track?.conversationId
  if (!trackConv) return false
  const currentId = conversationsStore.current?.id
  return !!currentId && String(currentId) === String(trackConv)
})

const visible = computed(() => !!audioStore.track && !onSourceConversation.value)

const progress = computed(() => {
  if (!audioStore.duration) return 0
  return Math.min(100, (audioStore.currentTime / audioStore.duration) * 100)
})

function format(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '00:00'
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = Math.floor(seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function goToConversation() {
  const tr = audioStore.track
  if (!tr || !tr.accountId || !tr.conversationId) return
  navigateTo(`/accounts/${tr.accountId}/conversations/${tr.conversationId}`)
}

function cycleSpeed() {
  const speeds = [1, 1.5, 2]
  const idx = speeds.indexOf(audioStore.playbackRate)
  const next = speeds[(idx + 1) % speeds.length] ?? 1
  audioStore.setPlaybackRate(next)
}
</script>

<template>
  <Transition
    enter-active-class="transition duration-200 ease-out"
    enter-from-class="translate-y-2 opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition duration-150 ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="translate-y-2 opacity-0"
  >
    <div
      v-if="visible"
      class="absolute inset-x-2 bottom-2 z-20 flex items-center gap-2 rounded-xl border border-primary/30 bg-elevated px-2.5 py-2 shadow-xl ring-1 ring-primary/20"
    >
      <UButton
        :icon="audioStore.isPlaying ? 'i-lucide-pause' : 'i-lucide-play'"
        color="primary"
        variant="solid"
        size="sm"
        class="shrink-0"
        :aria-label="audioStore.isPlaying ? t('conversations.audio.pause') : t('conversations.audio.play')"
        @click="audioStore.toggle()"
      />

      <button
        type="button"
        class="flex min-w-0 flex-1 flex-col items-start text-left"
        :disabled="!audioStore.track?.conversationId"
        @click="goToConversation"
      >
        <div class="flex w-full items-center gap-1.5">
          <UIcon name="i-lucide-audio-lines" class="size-3 shrink-0 text-primary" />
          <span class="truncate text-xs font-semibold text-highlighted">
            {{ audioStore.track?.title || t('conversations.audio.playing') }}
          </span>
        </div>
        <div class="mt-1 flex w-full items-center gap-1.5">
          <div class="h-1 flex-1 overflow-hidden rounded-full bg-default">
            <div class="h-full rounded-full bg-primary transition-[width] duration-100" :style="{ width: `${progress}%` }" />
          </div>
          <span class="shrink-0 font-mono text-[10px] tabular-nums text-muted">
            {{ format(audioStore.currentTime) }} / {{ format(audioStore.duration) }}
          </span>
        </div>
      </button>

      <button
        type="button"
        class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium leading-none tabular-nums text-muted ring ring-default transition-colors hover:bg-elevated"
        :aria-label="t('conversations.audio.speed')"
        @click.stop="cycleSpeed"
      >
        {{ audioStore.playbackRate }}x
      </button>

      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="xs"
        class="shrink-0"
        :aria-label="t('conversations.audio.stop')"
        @click="audioStore.stop()"
      />
    </div>
  </Transition>
</template>
