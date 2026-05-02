import { defineStore } from 'pinia'
import { markRaw, shallowRef } from 'vue'

export interface AudioTrack {
  id: string
  src?: string
  title?: string
  subtitle?: string
  accountId?: string | number
  conversationId?: string | number
}

export const useAudioPlayerStore = defineStore('audioPlayer', () => {
  const element = shallowRef<HTMLAudioElement | null>(null)
  const track = shallowRef<AudioTrack | null>(null)
  const isPlaying = ref(false)
  const currentTime = ref(0)
  const duration = ref(0)
  const playbackRate = ref(1)
  const muted = ref(false)

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

  async function play(next: AudioTrack) {
    const audio = ensureElement()
    const sameTrack = track.value?.id === next.id

    if (!sameTrack) {
      audio.pause()
      currentTime.value = 0
      duration.value = 0
      track.value = next
      if (!next.src) {
        isPlaying.value = false
        return
      }
      audio.src = next.src
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
