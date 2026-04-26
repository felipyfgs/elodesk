import { defineStore } from 'pinia'
import { markRaw, shallowRef } from 'vue'
import { useAuthStore } from '~/stores/auth'

export interface AudioTrack {
  id: string
  // Either a pre-resolved `src` (public URL / blob) OR `path` with `accountId`
  // for an authenticated download from MinIO. When both are given, `src`
  // wins (useful for the composer preview where the blob is already held).
  src?: string
  path?: string
  title?: string
  subtitle?: string
  accountId?: string | number
  conversationId?: string | number
}

// Global singleton audio — lives outside any conversation's DOM so playback
// survives navigation. Only one track plays at a time; starting a new one
// pauses and resets the previous. Owns its own authenticated blob URL so
// it keeps working even after the originating AudioPlayer unmounts.
export const useAudioPlayerStore = defineStore('audioPlayer', () => {
  const element = shallowRef<HTMLAudioElement | null>(null)
  const track = shallowRef<AudioTrack | null>(null)
  const isPlaying = ref(false)
  const currentTime = ref(0)
  const duration = ref(0)
  const playbackRate = ref(1)
  const muted = ref(false)
  let ownedBlobUrl: string | null = null

  function revokeBlob() {
    if (ownedBlobUrl) {
      URL.revokeObjectURL(ownedBlobUrl)
      ownedBlobUrl = null
    }
  }

  function ensureElement() {
    if (element.value) return element.value
    const audio = new Audio()
    audio.preload = 'auto'
    audio.addEventListener('play', () => {
      isPlaying.value = true
    })
    audio.addEventListener('pause', () => {
      isPlaying.value = false
    })
    audio.addEventListener('ended', () => {
      // WhatsApp-style auto-close: clear the track so the mini player
      // unmounts. Release the owned blob since the element won't be
      // replaying it.
      revokeBlob()
      isPlaying.value = false
      currentTime.value = 0
      duration.value = 0
      track.value = null
      audio.removeAttribute('src')
      audio.load()
    })
    audio.addEventListener('timeupdate', () => {
      currentTime.value = audio.currentTime
    })
    audio.addEventListener('loadedmetadata', () => {
      duration.value = Number.isFinite(audio.duration) ? audio.duration : 0
    })
    audio.addEventListener('error', () => {
      isPlaying.value = false
    })
    element.value = markRaw(audio)
    return audio
  }

  async function resolveSrc(next: AudioTrack): Promise<string | null> {
    if (next.src) return next.src
    if (!next.path || !next.accountId) return null
    const auth = useAuthStore()
    try {
      const runtime = useRuntimeConfig()
      const apiBase = runtime.public.apiUrl as string
      const url = `${apiBase}/accounts/${next.accountId}/uploads/download?path=${encodeURIComponent(next.path)}`
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${auth.accessToken}` }
      })
      if (!res.ok) throw new Error(`download failed: ${res.status}`)
      const blob = await res.blob()
      revokeBlob()
      ownedBlobUrl = URL.createObjectURL(blob)
      return ownedBlobUrl
    } catch (err) {
      console.error('[audioPlayer] resolveSrc failed', err)
      return null
    }
  }

  async function play(next: AudioTrack) {
    const audio = ensureElement()
    const sameTrack = track.value?.id === next.id

    if (!sameTrack) {
      audio.pause()
      currentTime.value = 0
      duration.value = 0
      track.value = next
      const src = await resolveSrc(next)
      if (!src) {
        isPlaying.value = false
        return
      }
      audio.src = src
      audio.currentTime = 0
      audio.playbackRate = playbackRate.value
      audio.muted = muted.value
    }

    try {
      await audio.play()
    } catch (err) {
      console.error('[audioPlayer] play failed', err)
      isPlaying.value = false
    }
  }

  function pause() {
    element.value?.pause()
  }

  function resume() {
    element.value?.play().catch(() => {
      isPlaying.value = false
    })
  }

  function toggle(id?: string) {
    if (!track.value) return
    if (id && track.value.id !== id) return
    if (isPlaying.value) pause()
    else resume()
  }

  function seek(time: number) {
    const audio = element.value
    if (!audio) return
    audio.currentTime = Math.max(0, Math.min(duration.value || audio.duration || 0, time))
    currentTime.value = audio.currentTime
  }

  function setPlaybackRate(rate: number) {
    playbackRate.value = rate
    if (element.value) element.value.playbackRate = rate
  }

  function setMuted(value: boolean) {
    muted.value = value
    if (element.value) element.value.muted = value
  }

  function stop() {
    const audio = element.value
    if (audio) {
      audio.pause()
      audio.removeAttribute('src')
      audio.load()
    }
    revokeBlob()
    track.value = null
    isPlaying.value = false
    currentTime.value = 0
    duration.value = 0
  }

  return {
    element,
    track,
    isPlaying,
    currentTime,
    duration,
    playbackRate,
    muted,
    play,
    pause,
    resume,
    toggle,
    seek,
    setPlaybackRate,
    setMuted,
    stop
  }
})
