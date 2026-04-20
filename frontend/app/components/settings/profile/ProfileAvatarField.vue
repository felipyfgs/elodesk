<script setup lang="ts">
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

const props = defineProps<{ modelValue?: string | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: string | null] }>()

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()

const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

async function onFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  if (!auth.user?.id || !auth.account?.id) return

  uploading.value = true
  try {
    const key = `${auth.account.id}/avatars/${auth.user.id}/${Date.now()}-${file.name}`
    const res = await api<{ uploadUrl: string, downloadUrl: string }>(
      `/accounts/${auth.account.id}/uploads/signed-url`,
      { method: 'POST', body: { key, contentType: file.type } }
    )
    await fetch(res.uploadUrl, { method: 'PUT', body: file, headers: { 'Content-Type': file.type } })
    emit('update:modelValue', key)
    toast.add({ title: t('settings.profile.avatar'), color: 'success' })
  } catch (err) {
    console.error(err)
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    uploading.value = false
  }
}

function openPicker() {
  fileInput.value?.click()
}

function clear() {
  emit('update:modelValue', null)
}
</script>

<template>
  <div class="flex items-center gap-4">
    <UAvatar :src="props.modelValue ?? undefined" size="xl" :alt="t('settings.profile.avatar')" />
    <div class="flex flex-col gap-2">
      <UButton variant="outline" :loading="uploading" @click="openPicker">
        {{ t('settings.profile.avatar') }}
      </UButton>
      <UButton
        v-if="props.modelValue"
        variant="ghost"
        color="error"
        size="xs"
        @click="clear"
      >
        {{ t('common.remove') }}
      </UButton>
      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        class="hidden"
        @change="onFileChange"
      >
    </div>
  </div>
</template>
