<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useSavedFiltersStore, type SavedFilter } from '~/stores/savedFilters'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

const props = defineProps<{
  modelValue: boolean
  query: FilterQueryPayload | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'saved': [filter: SavedFilter]
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const savedFilters = useSavedFiltersStore()
const toast = useToast()

const isOpen = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v)
})

const name = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

watch(isOpen, (v) => {
  if (!v) {
    name.value = ''
    error.value = null
  }
})

async function save() {
  if (!auth.account?.id || !props.query) return
  const trimmed = name.value.trim()
  if (!trimmed) {
    error.value = t('contacts.segments.nameRequired')
    return
  }

  submitting.value = true
  error.value = null
  try {
    const saved = await api<SavedFilter>(
      `/accounts/${auth.account.id}/custom_filters`,
      {
        method: 'POST',
        body: {
          name: trimmed,
          filter_type: 'contact',
          query: props.query
        }
      }
    )
    savedFilters.upsert(saved)
    toast.add({ title: t('contacts.segments.saved'), color: 'success' })
    emit('saved', saved)
    isOpen.value = false
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    error.value = e?.response?._data?.error ?? t('common.error')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="t('contacts.segments.saveTitle')"
    :description="t('contacts.segments.saveDescription')"
  >
    <template #body>
      <UFormField
        :label="t('contacts.segments.name')"
        :error="error ?? undefined"
      >
        <UInput
          v-model="name"
          :placeholder="t('contacts.segments.namePlaceholder')"
          class="w-full"
          @keydown.enter.prevent="save"
        />
      </UFormField>
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
          :loading="submitting"
          :disabled="!name.trim()"
          icon="i-lucide-save"
          @click="save"
        >
          {{ t('common.save') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
