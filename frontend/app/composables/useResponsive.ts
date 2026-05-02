import { breakpointsTailwind, useBreakpoints } from '@vueuse/core'

export function useResponsive() {
  const bp = useBreakpoints(breakpointsTailwind)

  const isMobile = bp.smaller('md')
  const isTablet = bp.between('md', 'lg')
  const isDesktop = bp.greaterOrEqual('lg')
  const isWide = bp.greaterOrEqual('xl')
  const isCompact = bp.smaller('lg')

  return { isMobile, isTablet, isDesktop, isWide, isCompact }
}
