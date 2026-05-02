import { useLocalStorage } from '@vueuse/core'
import { useResponsive } from '~/composables/useResponsive'

const STORAGE_KEY = 'elodesk:conversations:detailsOpen'

export function useDetailsSidebar() {
  const { isWide, isCompact } = useResponsive()

  const persisted = useLocalStorage<boolean | null>(STORAGE_KEY, null, {
    serializer: {
      read: (v: string) => (v === 'true' ? true : v === 'false' ? false : null),
      write: v => (v === null ? '' : String(v))
    }
  })

  const open = computed<boolean>({
    get: () => {
      if (persisted.value !== null) return persisted.value
      return isWide.value
    },
    set: (v) => { persisted.value = v }
  })

  function toggle() {
    open.value = !open.value
  }

  return { open, toggle, isCompact }
}
