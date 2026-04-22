<script setup lang="ts">
import { useLabelsStore, type Label } from '~/stores/labels'
import { useContactsStore } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'

const props = defineProps<{
  selectedIds: string[]
  visibleIds: string[]
  totalCount: number
}>()

const emit = defineEmits<{
  'select-all': []
  'clear-selection': []
  'delete-request': []
}>()

const { t } = useI18n()
const labelsStore = useLabelsStore()
const contactsStore = useContactsStore()
const auth = useAuthStore()
const toast = useToast()

const isOwnerAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const showLabelPicker = ref(false)
const selectedLabelId = ref<string | undefined>(undefined)

const selectedCount = computed(() => props.selectedIds.length)

const allVisibleSelected = computed(() => {
  if (!props.visibleIds.length) return false
  return props.visibleIds.every(id => props.selectedIds.includes(id))
})

function toggleSelectAll() {
  if (allVisibleSelected.value) {
    emit('clear-selection')
  } else {
    emit('select-all')
  }
}

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
  <Transition
    enter-active-class="transition-all duration-200"
    enter-from-class="opacity-0 -translate-y-2"
    enter-to-class="opacity-100 translate-y-0"
    leave-active-class="transition-all duration-200"
    leave-from-class="opacity-100 translate-y-0"
    leave-to-class="opacity-0 -translate-y-2"
  >
    <div
      v-if="selectedCount > 0"
      class="px-6 py-3 bg-elevated border-b border-default"
    >
      <div class="max-w-5xl mx-auto flex items-center justify-between gap-4">
        <div class="flex items-center gap-3">
          <UCheckbox
            :model-value="allVisibleSelected"
            :indeterminate="selectedCount > 0 && !allVisibleSelected"
            @update:model-value="toggleSelectAll"
          />
          <span class="text-sm font-medium">
            {{ t('contacts.bulk.selected', { count: selectedCount, total: totalCount }) }}
          </span>
        </div>

        <div class="flex items-center gap-2">
          <UButton
            :label="t('contacts.bulk.selectAll')"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="emit('select-all')"
          />

          <UDropdownMenu
            :items="[
              [
                { label: t('contacts.bulk.addLabel'), icon: 'i-lucide-tag-plus', onSelect: () => showLabelPicker = true },
                { label: t('contacts.bulk.removeLabel'), icon: 'i-lucide-tag-minus', onSelect: removeLabel }
              ],
              [
                { label: t('contacts.bulk.export'), icon: 'i-lucide-download', onSelect: exportCSV }
              ],
              ...(isOwnerAdmin
                ? [[{
                  label: t('contacts.bulk.delete'),
                  icon: 'i-lucide-trash',
                  color: 'error' as const,
                  onSelect: () => emit('delete-request')
                }]]
                : [])
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
            :label="t('contacts.bulk.clearSelection')"
            icon="i-lucide-x"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="emit('clear-selection')"
          />
        </div>
      </div>
    </div>
  </Transition>

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
