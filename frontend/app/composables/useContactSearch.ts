import type { Contact, ContactListResponse } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'

type IdentifierKind = 'phone' | 'email' | 'any'

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const PHONE_RE = /^\+?[\d\s\-()]{6,}$/

function detectKind(value: string): IdentifierKind {
  const v = value.trim()
  if (!v) return 'any'
  if (EMAIL_RE.test(v)) return 'email'
  if (PHONE_RE.test(v)) return 'phone'
  return 'any'
}

export function useContactSearch(identifierKind: Ref<IdentifierKind>) {
  const { t } = useI18n()
  const api = useApi()
  const auth = useAuthStore()
  const errorHandler = useErrorHandler()

  const searchTerm = ref('')
  const results = ref<Contact[]>([])
  const searching = ref(false)
  const selectedId = ref<string | undefined>(undefined)
  const creating = ref(false)

  const selected = computed(() =>
    results.value.find(c => c.id === selectedId.value) ?? null
  )

  let searchSeq = 0

  async function runSearch(term: string) {
    if (!auth.account?.id) return
    const seq = ++searchSeq
    searching.value = true
    try {
      const qs = term ? `?search=${encodeURIComponent(term)}&pageSize=20` : `?pageSize=30`
      const res = await api<ContactListResponse>(`/accounts/${auth.account.id}/contacts${qs}`)
      if (seq === searchSeq) results.value = res.payload
    } catch (err) {
      if (seq === searchSeq) errorHandler.handle(err)
    } finally {
      if (seq === searchSeq) searching.value = false
    }
  }

  async function loadRecent() {
    await runSearch('')
  }

  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  function clearDebounce() {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
  }

  function startDebounce(term: string, isActive: boolean) {
    clearDebounce()
    if (!isActive) return
    debounceTimer = setTimeout(() => runSearch(term), 300)
  }

  const items = computed(() =>
    results.value.map(c => ({
      label: c.name ?? c.email ?? c.phoneNumber ?? c.identifier ?? `#${c.id}`,
      description: [c.phoneNumber, c.email].filter(Boolean).join(' · '),
      icon: 'i-lucide-user',
      value: c.id
    }))
  )

  const canCreateFromTerm = computed(() => {
    const term = searchTerm.value.trim()
    if (!term || searching.value || creating.value || results.value.length > 0) return false
    const detected = detectKind(term)
    const expected = identifierKind.value
    if (expected === 'any') return detected !== 'any' || term.length >= 3
    return detected === expected
  })

  const createLabel = computed(() => {
    const term = searchTerm.value.trim()
    const detected = detectKind(term)
    if (detected === 'email') return t('contactsSendMessage.createWithEmail', { value: term })
    if (detected === 'phone') return t('contactsSendMessage.createWithPhone', { value: term })
    return t('contactsSendMessage.createWith', { value: term })
  })

  async function createFromTerm() {
    const term = searchTerm.value.trim()
    if (!term || !auth.account?.id) return
    creating.value = true
    try {
      const kind = detectKind(term)
      const payload: Record<string, string> = {}
      if (kind === 'email') {
        payload.email = term
        payload.name = term.split('@')[0] ?? term
      } else if (kind === 'phone') {
        payload.phone_number = term
        payload.name = term
      } else {
        payload.identifier = term
        payload.name = term
      }
      const created = await api<Contact>(`/accounts/${auth.account.id}/contacts`, {
        method: 'POST',
        body: payload
      })
      results.value = [created, ...results.value]
      selectedId.value = created.id
      searchTerm.value = ''
    } catch (err) {
      errorHandler.handle(err, { title: t('contactsSendMessage.createFailed') })
    } finally {
      creating.value = false
    }
  }

  function reset() {
    clearDebounce()
    searchSeq++
    searchTerm.value = ''
    results.value = []
    selectedId.value = undefined
  }

  function setInitial(contact: Contact) {
    const rest = results.value.filter(c => c.id !== contact.id)
    results.value = [contact, ...rest]
    selectedId.value = contact.id
  }

  return {
    searchTerm,
    results,
    searching,
    selectedId,
    creating,
    selected,
    items,
    canCreateFromTerm,
    createLabel,
    loadRecent,
    startDebounce,
    clearDebounce,
    createFromTerm,
    reset,
    setInitial
  }
}
