import { useLocalStorage } from '@vueuse/core'
import { useResponsive } from '~/composables/useResponsive'

// useDetailsSidebar centraliza o estado aberta/fechada da sidebar de detalhes
// na tela de conversas. A preferência é persistida em localStorage para
// sobreviver à navegação entre conversas e a recargas. O default depende da
// largura: em xl+ a sidebar começa aberta (espaço sobra), em telas menores
// começa fechada (Thread tem prioridade).
const STORAGE_KEY = 'elodesk:conversations:detailsOpen'

export function useDetailsSidebar() {
  const { isWide, isCompact } = useResponsive()

  // null = "ainda não decidido pelo usuário". Quando null, o getter abaixo
  // resolve para o default por viewport. Serializer explícito porque com
  // default `null` o VueUse cai no serializer string e devolve "true"/"false"
  // em vez de boolean — quebra a checagem de tipo em todos os consumidores.
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
