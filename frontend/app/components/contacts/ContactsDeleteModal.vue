<script setup lang="ts">
import { useContactsStore } from '~/stores/contacts'

const props = withDefaults(defineProps<{
  open: boolean
  count?: number
  ids?: string[]
}>(), {
  count: 0,
  ids: () => []
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  'deleted': []
}>()

const loading = ref(false)
const { t } = useI18n()
const errorHandler = useErrorHandler()
const contactsStore = useContactsStore()

const isOpen = computed({
  get: () => props.open,
  set: v => emit('update:open', v)
})

async function onSubmit() {
  if (!props.ids.length) return
  loading.value = true
  let success = 0
  let failure = 0
  try {
    for (const id of props.ids) {
      try {
        await contactsStore.remove(id)
        success++
      } catch (error) {
        failure++
        if (import.meta.dev) console.error(`[contacts] delete failed for ${id}`, error)
      }
    }
    if (failure === 0) {
      errorHandler.success(
        t('contacts.delete.bulk.success', { count: success })
      )
    } else {
      errorHandler.warning(
        t('contacts.delete.bulk.partial', { success, failure }),
        t('contacts.delete.bulk.partialDescription')
      )
    }
    emit('deleted')
    isOpen.value = false
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="t('contacts.delete.bulk.title', { count })"
    :description="t('contacts.delete.bulk.description')"
  >
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton
          :label="t('common.cancel')"
          color="neutral"
          variant="subtle"
          :disabled="loading"
          @click="isOpen = false"
        />
        <UButton
          :label="t('common.delete')"
          color="error"
          variant="solid"
          :loading="loading"
          :disabled="!ids.length"
          @click="onSubmit"
        />
      </div>
    </template>
  </UModal>
</template>
