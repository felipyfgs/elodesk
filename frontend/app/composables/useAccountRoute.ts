import { computed } from 'vue'

export function useAccountRoute() {
  const route = useRoute()

  const accountId = computed(() => route.params.accountId as string)

  return { accountId }
}
