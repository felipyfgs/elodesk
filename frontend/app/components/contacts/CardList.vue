<script setup lang="ts">
import type { Contact } from '~/stores/contacts'

const props = defineProps<{
  contacts: Contact[]
  selectedContactIds: string[]
  loading?: boolean
}>()

const emit = defineEmits<{
  toggleContact: [payload: { id: string, value: boolean }]
  updateContact: [payload: { id: string, name?: string, email?: string, phone_number?: string, additional_attributes?: Record<string, unknown> }]
  deleteContact: [id: string]
  showDetails: [id: string]
}>()

const expandedCardId = ref<string | null>(null)
const hoveredAvatarId = ref<string | null>(null)

const selectedIdsSet = computed(() => new Set(props.selectedContactIds))

function isSelected(id: string) {
  return selectedIdsSet.value.has(id)
}

function shouldShowSelection(id: string) {
  return hoveredAvatarId.value === id || isSelected(id)
}

function toggleExpanded(id: string) {
  expandedCardId.value = expandedCardId.value === id ? null : id
}

function handleSelect(id: string, value: boolean) {
  emit('toggleContact', { id, value })
}

function handleUpdate(payload: { id: string, name?: string, email?: string, phone_number?: string, additional_attributes?: Record<string, unknown> }) {
  emit('updateContact', payload)
  // Fechar card após atualização
  expandedCardId.value = null
}

function handleDelete(id: string) {
  emit('deleteContact', id)
}

function handleShowDetails(id: string) {
  emit('showDetails', id)
}

function handleAvatarHover(id: string, isHovered: boolean) {
  hoveredAvatarId.value = isHovered ? id : null
}
</script>

<template>
  <div class="flex flex-col gap-1.5 px-4 py-3 max-w-5xl mx-auto w-full">
    <ContactsCard
      v-for="contact in contacts"
      :key="contact.id"
      :contact="contact"
      :is-expanded="expandedCardId === contact.id"
      :is-selected="isSelected(contact.id)"
      :selectable="shouldShowSelection(contact.id)"
      :loading="loading"
      @toggle="toggleExpanded(contact.id)"
      @select="(value) => handleSelect(contact.id, value)"
      @update="handleUpdate"
      @delete="handleDelete"
      @show-details="handleShowDetails"
      @mouseenter="handleAvatarHover(contact.id, true)"
      @mouseleave="handleAvatarHover(contact.id, false)"
    />
  </div>
</template>
