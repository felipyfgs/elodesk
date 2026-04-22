<script setup lang="ts">
import type { Contact, ContactListResponse } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { useContactsStore } from '~/stores/contacts'

const props = defineProps<{
  modelValue: boolean
  child: Contact
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'merged': [primary: Contact]
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()
const toast = useToast()

const isOpen = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v)
})

const searchTerm = ref('')
const searchResults = ref<Contact[]>([])
const searching = ref(false)
const selectedPrimaryId = ref<string | undefined>(undefined)
const submitting = ref(false)

const selectedPrimary = computed(() =>
  searchResults.value.find(c => c.id === selectedPrimaryId.value) ?? null
)

let debounce: ReturnType<typeof setTimeout> | null = null

watch(searchTerm, (val) => {
  if (debounce) clearTimeout(debounce)
  if (!val || val.length < 2) {
    searchResults.value = []
    return
  }
  debounce = setTimeout(async () => {
    if (!auth.account?.id) return
    searching.value = true
    try {
      const res = await api<ContactListResponse>(
        `/accounts/${auth.account.id}/contacts?search=${encodeURIComponent(val)}&pageSize=20`
      )
      searchResults.value = res.payload.filter(c => c.id !== props.child.id)
    } finally {
      searching.value = false
    }
  }, 300)
})

watch(isOpen, (v) => {
  if (!v) {
    searchTerm.value = ''
    searchResults.value = []
    selectedPrimaryId.value = undefined
  }
})

function initials(c: Contact): string {
  return (c.name ?? '?')
    .split(' ')
    .map(w => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

const selectItems = computed(() =>
  searchResults.value.map(c => ({
    label: c.name ?? c.email ?? c.identifier ?? `#${c.id}`,
    value: c.id
  }))
)

async function merge() {
  if (!selectedPrimaryId.value) return
  submitting.value = true
  try {
    const primary = await contactsStore.merge(props.child.id, selectedPrimaryId.value)
    if (primary) {
      toast.add({ title: t('contacts.merge.success'), color: 'success' })
      emit('merged', primary)
      isOpen.value = false
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
  <UModal v-model:open="isOpen" :title="t('contacts.merge.title')" :description="t('contacts.merge.description')">
    <template #body>
      <div class="space-y-4">
        <div class="rounded-lg border border-error/40 p-3 space-y-1">
          <p class="text-xs font-medium text-error uppercase">
            {{ t('contacts.merge.child') }}
          </p>
          <div class="flex items-center gap-3">
            <UAvatar :text="initials(child)" size="md" />
            <div class="min-w-0">
              <p class="font-medium truncate">
                {{ child.name || '—' }}
              </p>
              <p class="text-xs text-muted truncate">
                {{ child.email ?? child.phoneNumber ?? child.identifier ?? '' }}
              </p>
            </div>
          </div>
        </div>

        <UFormField :label="t('contacts.merge.searchLabel')">
          <USelectMenu
            v-model="selectedPrimaryId"
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

        <div v-if="selectedPrimary" class="rounded-lg border border-success/40 p-3 space-y-1">
          <p class="text-xs font-medium text-success uppercase">
            {{ t('contacts.merge.primary') }}
          </p>
          <div class="flex items-center gap-3">
            <UAvatar :text="initials(selectedPrimary)" size="md" />
            <div class="min-w-0">
              <p class="font-medium truncate">
                {{ selectedPrimary.name || '—' }}
              </p>
              <p class="text-xs text-muted truncate">
                {{ selectedPrimary.email ?? selectedPrimary.phoneNumber ?? selectedPrimary.identifier ?? '' }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="submitting"
          @click="isOpen = false"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          color="primary"
          :loading="submitting"
          :disabled="!selectedPrimaryId"
          icon="i-lucide-git-merge"
          @click="merge"
        >
          {{ t('contacts.merge.submit') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
