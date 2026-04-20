<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Contact } from '~/stores/contacts'
import { contactCreateSchema, type ContactCreateForm } from '~/schemas/contacts'

const props = defineProps<{
  contact: Contact
  contactId: string
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const saving = ref(false)
const errors = ref<Record<string, string>>({})

const form = reactive<ContactCreateForm>({
  name: props.contact.name ?? '',
  email: props.contact.email ?? null,
  phone_number: props.contact.phoneNumber ?? null,
  identifier: props.contact.identifier ?? null
})

async function saveContact() {
  const result = contactCreateSchema.safeParse(form)
  if (!result.success) {
    errors.value = Object.fromEntries(result.error.issues.map(issue => [issue.path.join('.'), issue.message]))
    return
  }
  errors.value = {}
  saving.value = true
  try {
    await api(`/accounts/${auth.account!.id}/contacts/${props.contactId}`, {
      method: 'POST',
      body: result.data
    })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <UPageCard variant="outline" :title="t('contactDetail.title')">
      <form class="flex flex-col gap-4" @submit.prevent="saveContact">
        <UFormField :label="t('contactDetail.name')" :error="errors.name">
          <UInput v-model="form.name" class="w-full" />
        </UFormField>

        <UFormField :label="t('contactDetail.email')" :error="errors.email">
          <UInput v-model="form.email!" type="email" class="w-full" />
        </UFormField>

        <UFormField :label="t('contactDetail.phone')" :error="errors.phone_number">
          <UInput v-model="form.phone_number!" class="w-full" />
        </UFormField>

        <UFormField :label="t('contactDetail.identifier')" :error="errors.identifier">
          <UInput v-model="form.identifier!" class="w-full" />
        </UFormField>

        <div class="flex justify-end gap-2">
          <UButton type="submit" :loading="saving">
            {{ t('contactDetail.save') }}
          </UButton>
        </div>
      </form>
    </UPageCard>

    <UPageCard variant="outline" :title="t('contactDetail.labels')">
      <LabelPicker :contact-id="contactId" />
    </UPageCard>

    <ContactCustomAttributes
      :contact-id="contactId"
      :values="(contact.additionalAttributes ? JSON.parse(contact.additionalAttributes) : {}) as Record<string, unknown>"
    />
  </div>
</template>
