<script setup lang="ts">
import { useLabelsStore, type Label } from '~/stores/labels'
import { useContactsStore } from '~/stores/contacts'

const props = defineProps<{
  selectedIds: string[]
}>()

const emit = defineEmits<{
  'clear-selection': []
}>()

const { t } = useI18n()
const labelsStore = useLabelsStore()
const contactsStore = useContactsStore()
const toast = useToast()

const showLabelPicker = ref(false)
const selectedLabelId = ref<string | undefined>(undefined)

const selectedCount = computed(() => props.selectedIds.length)

async function addLabel() {
  if (!selectedLabelId.value || !selectedCount.value) return
  await contactsStore.applyBulkLabel(props.selectedIds, selectedLabelId.value, 'add')
  toast.add({ title: `${selectedCount.value} contacts updated`, color: 'success' })
  showLabelPicker.value = false
  selectedLabelId.value = undefined
  emit('clear-selection')
}

async function removeLabel() {
  if (!selectedLabelId.value || !selectedCount.value) return
  await contactsStore.applyBulkLabel(props.selectedIds, selectedLabelId.value, 'remove')
  toast.add({ title: `${selectedCount.value} contacts updated`, color: 'success' })
  showLabelPicker.value = false
  selectedLabelId.value = undefined
  emit('clear-selection')
}

function exportCSV() {
  // Export selected contacts as CSV
  const contacts = contactsStore.list.filter(c => props.selectedIds.includes(c.id))
  const header = 'name,email,phone_number\n'
  const rows = contacts.map(c =>
    `"${c.name ?? ''}","${c.email ?? ''}","${c.phoneNumber ?? ''}"`
  ).join('\n')
  const blob = new Blob([header + rows], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'contacts.csv'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div v-if="selectedCount > 0" class="flex items-center gap-2">
    <UBadge variant="subtle" color="primary">
      {{ selectedCount }} selected
    </UBadge>

    <UDropdownMenu
      :items="[
        [
          { label: t('contacts.bulk.addLabel'), icon: 'i-lucide-tag-plus', onSelect: () => showLabelPicker = true },
          { label: t('contacts.bulk.removeLabel'), icon: 'i-lucide-tag-minus', onSelect: removeLabel }
        ],
        [
          { label: t('contacts.bulk.export'), icon: 'i-lucide-download', onSelect: exportCSV }
        ]
      ]"
    >
      <UButton
        icon="i-lucide-ellipsis-vertical"
        color="neutral"
        variant="outline"
        size="sm"
      />
    </UDropdownMenu>

    <UButton
      icon="i-lucide-x"
      color="neutral"
      variant="ghost"
      size="sm"
      @click="emit('clear-selection')"
    />
  </div>

  <!-- Label picker modal -->
  <UModal v-model:open="showLabelPicker" :title="t('contacts.bulk.addLabel')">
    <template #body>
      <div class="space-y-3">
        <USelect
          v-model="selectedLabelId"
          :items="labelsStore.list.map((l: Label) => ({ label: l.title, value: l.id }))"
          placeholder="Select a label…"
          class="w-full"
        />
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="showLabelPicker = false">
            {{ t('common.cancel') }}
          </UButton>
          <UButton :disabled="!selectedLabelId" @click="addLabel">
            {{ t('common.save') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
