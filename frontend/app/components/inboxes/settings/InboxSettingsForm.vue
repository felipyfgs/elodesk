<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Inbox } from '~/stores/inboxes'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const showRotateModal = ref(false)
const rotating = ref(false)
const newToken = ref('')
const showToken = ref(false)

async function rotateHmac() {
  if (!auth.account?.id) return
  rotating.value = true
  try {
    const res = await api<{ hmacToken: string }>(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}/rotate_hmac`, {
      method: 'POST'
    })
    newToken.value = res.hmacToken
    showToken.value = true
    toast.add({ title: t('inboxes.rotateHmacSuccess'), color: 'success', duration: 10000 })
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    rotating.value = false
  }
}

async function copyToken() {
  try {
    await navigator.clipboard.writeText(newToken.value)
    toast.add({ title: t('common.copied'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  }
}

function closeModal() {
  showRotateModal.value = false
  newToken.value = ''
  showToken.value = false
}
</script>

<template>
  <div>
    <UButton
      v-if="inbox.channelType === 'Channel::WebWidget'"
      icon="i-lucide-refresh-cw"
      variant="outline"
      color="warning"
      size="sm"
      @click="showRotateModal = true"
    >
      {{ t('inboxes.rotateHmac') }}
    </UButton>

    <UModal v-model:open="showRotateModal">
      <template #content>
        <div class="p-6">
          <template v-if="!showToken">
            <h3 class="text-lg font-semibold">
              {{ t('inboxes.rotateHmac') }}
            </h3>
            <p class="mt-2 text-sm text-muted">
              {{ t('inboxes.rotateHmacConfirm') }}
            </p>
            <div class="flex justify-end gap-2 mt-6">
              <UButton variant="ghost" color="neutral" @click="closeModal">
                {{ t('common.cancel') }}
              </UButton>
              <UButton color="warning" :loading="rotating" @click="rotateHmac">
                {{ t('inboxes.rotateHmac') }}
              </UButton>
            </div>
          </template>

          <template v-else>
            <h3 class="text-lg font-semibold text-warning">
              {{ t('inboxes.rotateHmacWarning') }}
            </h3>
            <p class="mt-2 text-sm text-muted">
              {{ t('inboxes.rotateHmacCopyWarning') }}
            </p>
            <div class="mt-4 rounded-lg bg-[var(--ui-bg-accented)] p-3">
              <code class="text-xs break-all font-mono">{{ newToken }}</code>
            </div>
            <div class="flex justify-end gap-2 mt-6">
              <UButton icon="i-lucide-copy" @click="copyToken">
                {{ t('inboxes.secret.copy') }}
              </UButton>
              <UButton @click="closeModal">
                {{ t('common.close') }}
              </UButton>
            </div>
          </template>
        </div>
      </template>
    </UModal>
  </div>
</template>
