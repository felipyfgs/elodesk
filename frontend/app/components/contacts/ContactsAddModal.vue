<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { contactCreateSchema, type ContactCreateForm } from '~/schemas/contacts'
import { useContactsStore, type Contact } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'

const open = ref(false)
const toast = useToast()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()

const state = reactive<ContactCreateForm>({
  name: '',
  email: null,
  phone_number: null,
  identifier: null
})

const loading = ref(false)

async function onSubmit(event: FormSubmitEvent<ContactCreateForm>) {
  loading.value = true
  try {
    const res = await api<Contact>(`/accounts/${auth.account!.id}/contacts`, {
      method: 'POST',
      body: {
        name: event.data.name,
        email: event.data.email,
        phone_number: event.data.phone_number,
        identifier: event.data.identifier,
        source_id: 'manual'
      }
    })
    contactsStore.upsert(res)
    toast.add({ title: 'Contact created', description: `${event.data.name} added successfully`, color: 'success' })
    open.value = false
    state.name = ''
    state.email = null
    state.phone_number = null
    state.identifier = null
  } catch {
    toast.add({ title: 'Error creating contact', color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="$t('contacts.new')" description="Add a new contact">
    <UButton label="New contact" icon="i-lucide-plus" />

    <template #body>
      <UForm
        :schema="contactCreateSchema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField :label="$t('contactDetail.name')" name="name" required>
          <UInput v-model="state.name" class="w-full" />
        </UFormField>
        <UFormField :label="$t('contactDetail.email')" name="email">
          <UInput v-model="state.email!" type="email" class="w-full" />
        </UFormField>
        <UFormField :label="$t('contactDetail.phone')" name="phone_number">
          <UInput v-model="state.phone_number!" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            :label="$t('common.cancel')"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            :label="$t('common.create')"
            color="primary"
            variant="solid"
            type="submit"
            :loading="loading"
          />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
