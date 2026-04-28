<script setup lang="ts">
import { formatDistanceToNow } from 'date-fns'
import { ConfirmModal } from '#components'
import type { Contact } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { useContactsStore } from '~/stores/contacts'
import type { ContactFormState } from './Form.vue'

const props = defineProps<{
  contact: Contact
}>()

const emit = defineEmits<{
  updated: [contact: Contact]
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()
const toast = useToast()
const confirm = useOverlay().create(ConfirmModal)

const formRef = ref<{ submit: () => void, isInvalid: boolean } | null>(null)
const saving = ref(false)

const isOwnerAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const createdLabel = computed(() => {
  if (!props.contact.createdAt) return ''
  return formatDistanceToNow(new Date(props.contact.createdAt), { addSuffix: true })
})

const lastActivityLabel = computed(() => {
  if (!props.contact.lastActivityAt) return ''
  return formatDistanceToNow(new Date(props.contact.lastActivityAt), { addSuffix: true })
})

async function onSubmit(state: ContactFormState) {
  if (!auth.account?.id) return
  saving.value = true
  try {
    const name = `${state.firstName} ${state.lastName}`.trim()
    const body = {
      name,
      email: state.email || null,
      phone_number: state.phoneNumber || null,
      additional_attributes: {
        city: state.city || undefined,
        country: state.country || undefined,
        description: state.bio || undefined,
        companyName: state.companyName || undefined,
        socialProfiles: state.socialProfiles
      }
    }
    const updated = await api<Contact>(
      `/accounts/${auth.account.id}/contacts/${props.contact.id}`,
      { method: 'POST', body }
    )
    contactsStore.upsert(updated)
    emit('updated', updated)
    toast.add({ title: t('contacts.card.updateSuccess'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    toast.add({ title: e?.response?._data?.error ?? t('contacts.card.updateError'), color: 'error' })
  } finally {
    saving.value = false
  }
}

function submit() {
  formRef.value?.submit()
}

async function confirmDelete() {
  const confirmed = await confirm.open({
    title: t('contacts.delete.one.title'),
    description: t('contacts.delete.one.description', { name: props.contact.name || props.contact.email || '' }),
    confirmColor: 'error',
    confirmLabel: t('common.delete')
  }).result
  if (!confirmed) return
  try {
    await contactsStore.remove(props.contact.id)
    toast.add({ title: t('contacts.delete.success'), color: 'success' })
    await navigateTo(auth.account?.id ? `/accounts/${auth.account.id}/contacts` : '/contacts')
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    toast.add({ title: e?.response?._data?.error ?? t('common.error'), color: 'error' })
  }
}
</script>

<template>
  <div class="flex flex-col gap-8 pb-6">
    <!-- Header: avatar + name + metadata -->
    <div class="flex flex-col items-start gap-3">
      <ContactsAvatarUploader :contact="contact" />

      <div class="flex items-center gap-2">
        <h2 class="text-lg font-semibold text-highlighted">
          {{ contact.name || '—' }}
        </h2>
        <UBadge
          v-if="contact.blocked"
          color="error"
          variant="subtle"
          size="sm"
        >
          {{ t('contacts.actions.blocked') }}
        </UBadge>
      </div>

      <div class="flex flex-col gap-1 text-sm text-muted">
        <span v-if="contact.identifier" class="inline-flex items-center gap-1.5">
          <UIcon name="i-lucide-hash" class="size-4 text-dimmed" />
          {{ contact.identifier }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <UIcon name="i-lucide-activity" class="size-4 text-dimmed" />
          {{ t('contactDetail.meta.createdAt', { date: createdLabel }) }}
          <template v-if="lastActivityLabel">
            · {{ t('contactDetail.meta.lastActivity', { date: lastActivityLabel }) }}
          </template>
        </span>
      </div>

      <LabelPicker :contact-id="String(contact.id)" />
    </div>

    <!-- Form -->
    <ContactsDetailForm
      ref="formRef"
      :contact="contact"
      @submit="onSubmit"
    />

    <div>
      <UButton
        :label="t('contacts.card.updateButton')"
        color="primary"
        size="sm"
        :loading="saving"
        @click="submit"
      />
    </div>

    <!-- Danger zone -->
    <div
      v-if="isOwnerAdmin"
      class="flex flex-col gap-3 pt-6 border-t border-default"
    >
      <div>
        <h4 class="text-sm font-medium text-highlighted">
          {{ t('contactDetail.delete.title') }}
        </h4>
        <p class="text-sm text-muted mt-1">
          {{ t('contactDetail.delete.description') }}
        </p>
      </div>
      <div>
        <UButton
          color="error"
          variant="soft"
          size="sm"
          icon="i-lucide-trash-2"
          :label="t('contactDetail.delete.action')"
          @click="confirmDelete"
        />
      </div>
    </div>
  </div>
</template>
