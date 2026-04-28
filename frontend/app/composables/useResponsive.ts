import { breakpointsTailwind, useBreakpoints } from '@vueuse/core'

// Composable central de breakpoints. Espelha os tokens do Tailwind (sm 640,
// md 768, lg 1024, xl 1280, 2xl 1536) e nomeia os modos que efetivamente
// usamos no app — assim o restante do código não importa breakpoints crus.
//
//   isMobile  → menor que md (640..767 e abaixo): celular em retrato.
//   isTablet  → md a lg (768..1023): tablet, laptop pequeno, ou DevTools aberta.
//   isDesktop → ≥ lg: layout 2 colunas (lista + thread) cabe.
//   isWide    → ≥ xl: layout 3 colunas (lista + thread + sidebar) cabe à vontade.
//
// Use `isCompact` quando quiser "qualquer coisa abaixo de lg" (mobile + tablet).
export function useResponsive() {
  const bp = useBreakpoints(breakpointsTailwind)

  const isMobile = bp.smaller('md')
  const isTablet = bp.between('md', 'lg')
  const isDesktop = bp.greaterOrEqual('lg')
  const isWide = bp.greaterOrEqual('xl')
  const isCompact = bp.smaller('lg')

  return { isMobile, isTablet, isDesktop, isWide, isCompact }
}
