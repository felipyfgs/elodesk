<script setup lang="ts">
import type { Contact } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { useContactsStore } from '~/stores/contacts'

const props = defineProps<{
  contact: Contact
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()
const toast = useToast()

const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const deleting = ref(false)
const currentObjectKey = ref<string | null>(props.contact.avatarUrl)
const signedAvatarUrl = ref<string>()
let avatarRequestSeq = 0

const initials = computed(() => {
  const name = props.contact.name ?? ''
  return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2) || '?'
})

const avatarUrl = computed(() => signedAvatarUrl.value)

async function refreshSignedAvatar() {
  const objectKey = currentObjectKey.value
  if (!objectKey || !auth.account?.id) {
    signedAvatarUrl.value = undefined
    return
  }

  if (/^https?:\/\//.test(objectKey)) {
    signedAvatarUrl.value = objectKey
    return
  }

  const requestSeq = ++avatarRequestSeq
  try {
    const { download_url: downloadURL } = await api<{ download_url: string }>(
      `/accounts/${auth.account.id}/uploads/signed-url?path=${encodeURIComponent(objectKey)}`
    )
    if (requestSeq === avatarRequestSeq) {
      signedAvatarUrl.value = downloadURL
    }
  } catch (err) {
    if (requestSeq === avatarRequestSeq) {
      signedAvatarUrl.value = undefined
    }
    if (import.meta.dev) console.warn('[contacts] failed to sign avatar url', err)
  }
}

watch(
  () => props.contact.avatarUrl,
  (value) => {
    currentObjectKey.value = value
    void refreshSignedAvatar()
  },
  { immediate: true }
)

function trigger() {
  fileInput.value?.click()
}

async function onFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.add({ title: t('contacts.avatar.invalidType'), color: 'error' })
    target.value = ''
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    toast.add({ title: t('contacts.avatar.tooLarge'), color: 'error' })
    target.value = ''
    return
  }

  uploading.value = true
  try {
    const updated = await contactsStore.uploadAvatar(props.contact.id, file)
    currentObjectKey.value = updated.avatarUrl
    await refreshSignedAvatar()
    toast.add({ title: t('contacts.avatar.uploaded'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    toast.add({ title: e?.response?._data?.error ?? t('common.error'), color: 'error' })
  } finally {
    uploading.value = false
    target.value = ''
  }
}

async function removeAvatar() {
  deleting.value = true
  try {
    await contactsStore.deleteAvatar(props.contact.id)
    currentObjectKey.value = null
    signedAvatarUrl.value = undefined
    toast.add({ title: t('contacts.avatar.removed'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { response?: { _data?: { error?: string } } }
    toast.add({ title: e?.response?._data?.error ?? t('common.error'), color: 'error' })
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="relative inline-block">
    <UAvatar
      :text="initials"
      :src="avatarUrl"
      size="2xl"
      class="shrink-0"
    />

    <input
      ref="fileInput"
      type="file"
      accept="image/*"
      class="hidden"
      @change="onFileChange"
    >

    <div class="absolute -bottom-1 -right-1 flex gap-1">
      <UButton
        icon="i-lucide-camera"
        color="primary"
        size="xs"
        :loading="uploading"
        :aria-label="t('contacts.avatar.upload')"
        @click="trigger"
      />
      <UButton
        v-if="contact.avatarUrl"
        icon="i-lucide-x"
        color="error"
        size="xs"
        variant="soft"
        :loading="deleting"
        :aria-label="t('contacts.avatar.delete')"
        @click="removeAvatar"
      />
    </div>
  </div>
</template>
