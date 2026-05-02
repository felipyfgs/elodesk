<script setup lang="ts">
import type { Contact } from '~/stores/contacts'
import { computed, ref, watch } from 'vue'
import { useContactSearch } from '~/composables/useContactSearch'
import type { LinkedEntityType } from '~/schemas/pipeline'

const props = defineProps<{
  open: boolean
  kind: 'contact' | 'conversation'
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'picked': [payload: { type: LinkedEntityType, id: number, label: string }]
}>()

const { t } = useI18n()
const errorHandler = useErrorHandler()

// === Contact picker (uses existing composable) ===
const identifierKind = ref<'any'>('any')
const {
  searchTerm,
  searching,
  selectedId: selectedContactId,
  selected: selectedContact,
  items: contactItems,
  loadRecent,
  startDebounce,
  reset: resetSearch
} = useContactSearch(identifierKind)

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    resetSearch()
    if (props.kind === 'contact') void loadRecent()
  }
})

watch(searchTerm, (term) => {
  if (props.open && props.kind === 'contact') startDebounce(term, true)
})

// === Conversation picker (manual ID input as MVP) ===
const conversationId = ref<string>('')

function pickContact() {
  if (!selectedContact.value) return
  const c = selectedContact.value as Contact
  emit('picked', {
    type: 'contact',
    id: Number(c.id),
    label: c.name ?? c.email ?? c.phoneNumber ?? `#${c.id}`
  })
  emit('update:open', false)
}

function pickConversation() {
  const id = Number(conversationId.value)
  if (!id || Number.isNaN(id)) {
    errorHandler.warning(t('pipelines.link.invalidConversation'))
    return
  }
  emit('picked', { type: 'conversation', id, label: `#${id}` })
  emit('update:open', false)
}

const canConfirm = computed(() => {
  if (props.kind === 'contact') return !!selectedContactId.value
  return !!conversationId.value.trim()
})

function confirm() {
  if (!canConfirm.value) return
  if (props.kind === 'contact') pickContact()
  else pickConversation()
}
</script>

<template>
  <UModal
    :open="props.open"
    :title="props.kind === 'contact' ? t('pipelines.link.pickContact') : t('pipelines.link.pickConversation')"
    :ui="{ content: 'sm:max-w-md' }"
    @update:open="value => emit('update:open', value)"
  >
    <template #body>
      <div v-if="props.kind === 'contact'" class="flex flex-col gap-3">
        <UInput
          v-model="searchTerm"
          :placeholder="t('pipelines.link.searchContact')"
          icon="i-lucide-search"
          :loading="searching"
          autofocus
        />

        <div class="max-h-80 overflow-auto flex flex-col gap-1">
          <button
            v-for="item in contactItems"
            :key="item.value"
            type="button"
            class="text-left p-2 rounded hover:bg-elevated/50 transition-colors flex items-center gap-2"
            :class="selectedContactId === item.value ? 'bg-elevated/70' : ''"
            @click="selectedContactId = item.value"
          >
            <UIcon name="i-lucide-user" class="size-4 text-muted shrink-0" />
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium truncate">
                {{ item.label }}
              </p>
              <p v-if="item.description" class="text-xs text-muted truncate">
                {{ item.description }}
              </p>
            </div>
            <UIcon
              v-if="selectedContactId === item.value"
              name="i-lucide-check"
              class="size-4 text-primary"
            />
          </button>
          <p v-if="!searching && !contactItems.length" class="text-xs text-muted text-center py-4">
            {{ t('pipelines.link.noContacts') }}
          </p>
        </div>
      </div>

      <div v-else class="flex flex-col gap-3">
        <p class="text-sm text-muted">
          {{ t('pipelines.link.conversationHelp') }}
        </p>
        <UFormField :label="t('pipelines.link.conversationId')">
          <UInput v-model="conversationId" type="number" placeholder="123" autofocus />
        </UFormField>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton color="neutral" variant="ghost" @click="emit('update:open', false)">
          {{ t('common.cancel') }}
        </UButton>
        <UButton :disabled="!canConfirm" @click="confirm">
          {{ t('common.confirm') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
