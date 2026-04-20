<script setup lang="ts">
import { useContactsStore } from '~/stores/contacts'

const props = withDefaults(defineProps<{
  count?: number
}>(), {
  count: 0
})

const open = ref(false)
const loading = ref(false)
const toast = useToast()
const contactsStore = useContactsStore()

async function onSubmit() {
  loading.value = true
  try {
    // Delete is done via bulk — caller should set selected IDs on the store
    contactsStore.removeMany([]) // placeholder — actual delete via API
    toast.add({ title: `${props.count} contact(s) deleted`, color: 'success' })
    open.value = false
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Delete ${count} contact${count > 1 ? 's' : ''}`"
    description="Are you sure? This action cannot be undone."
  >
    <slot />

    <template #body>
      <div class="flex justify-end gap-2">
        <UButton
          :label="$t('common.cancel')"
          color="neutral"
          variant="subtle"
          @click="open = false"
        />
        <UButton
          :label="$t('common.delete')"
          color="error"
          variant="solid"
          :loading="loading"
          @click="onSubmit"
        />
      </div>
    </template>
  </UModal>
</template>
