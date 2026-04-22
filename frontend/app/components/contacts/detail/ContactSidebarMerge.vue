<script setup lang="ts">
import type { Contact, ContactListResponse } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { useContactsStore } from '~/stores/contacts'

const props = defineProps<{
  contact: Contact
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()
const toast = useToast()

const searchTerm = ref('')
const results = ref<Contact[]>([])
const searching = ref(false)
const selectedId = ref<string | undefined>()
const submitting = ref(false)

const selected = computed(() =>
  results.value.find(c => c.id === selectedId.value) ?? null
)

let debounce: ReturnType<typeof setTimeout> | null = null

watch(searchTerm, (val) => {
  if (debounce) clearTimeout(debounce)
  if (!val || val.length < 2) {
    results.value = []
    return
  }
  debounce = setTimeout(async () => {
    if (!auth.account?.id) return
    searching.value = true
    try {
      const res = await api<ContactListResponse>(
        `/accounts/${auth.account.id}/contacts?search=${encodeURIComponent(val)}&pageSize=20`
      )
      results.value = res.payload.filter(c => c.id !== props.contact.id)
    } finally {
      searching.value = false
    }
  }, 300)
})

const selectItems = computed(() =>
  results.value.map(c => ({
    label: c.name ?? c.email ?? c.identifier ?? `#${c.id}`,
    value: c.id
  }))
)

function initials(c: Contact): string {
  return (c.name ?? '?')
    .split(' ')
    .map(w => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

async function merge() {
  if (!selectedId.value) return
  submitting.value = true
  try {
    const primary = await contactsStore.merge(props.contact.id, selectedId.value)
    if (primary) {
      toast.add({ title: t('contacts.merge.success'), color: 'success' })
      await navigateTo(auth.account?.id ? `/accounts/${auth.account.id}/contacts/${primary.id}` : `/contacts/${primary.id}`)
    }
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    toast.add({ title: e?.response?._data?.error ?? t('common.error'), color: 'error' })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-medium text-highlighted">
      {{ t('contactDetail.sidebar.merge') }}
    </h3>
    <p class="text-xs text-muted">
      {{ t('contacts.merge.description') }}
    </p>

    <div class="rounded-md border border-error/40 bg-error/5 p-3 flex flex-col gap-1">
      <p class="text-xs font-medium text-error uppercase">
        {{ t('contacts.merge.child') }}
      </p>
      <div class="flex items-center gap-2">
        <UAvatar :text="initials(contact)" size="sm" />
        <p class="text-sm font-medium truncate">
          {{ contact.name || '—' }}
        </p>
      </div>
    </div>

    <UFormField :label="t('contacts.merge.searchLabel')">
      <USelectMenu
        v-model="selectedId"
        :items="selectItems"
        value-key="value"
        searchable
        :searchable-placeholder="t('contacts.merge.searchPlaceholder')"
        :placeholder="t('contacts.merge.selectPrimary')"
        :loading="searching"
        class="w-full"
        @update:search-term="searchTerm = $event"
      />
    </UFormField>

    <div v-if="selected" class="rounded-md border border-success/40 bg-success/5 p-3 flex flex-col gap-1">
      <p class="text-xs font-medium text-success uppercase">
        {{ t('contacts.merge.primary') }}
      </p>
      <div class="flex items-center gap-2">
        <UAvatar :text="initials(selected)" size="sm" />
        <p class="text-sm font-medium truncate">
          {{ selected.name || '—' }}
        </p>
      </div>
    </div>

    <UButton
      color="primary"
      icon="i-lucide-git-merge"
      :loading="submitting"
      :disabled="!selectedId"
      :label="t('contacts.merge.submit')"
      @click="merge"
    />
  </div>
</template>
