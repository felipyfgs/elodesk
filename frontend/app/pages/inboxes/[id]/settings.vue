<script setup lang="ts">
import type { Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const name = ref(props.inbox.name)
const saving = ref(false)

async function saveName() {
  if (!auth.account?.id) return
  saving.value = true
  try {
    await api(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}`, {
      method: 'PUT',
      body: { name: name.value }
    })
    toast.add({ title: t('common.success'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <UPageCard :title="t('inboxes.general')" variant="subtle">
      <UForm class="flex flex-col gap-4" @submit="saveName">
        <UFormField :label="t('inboxes.wizards.name')" name="name">
          <UInput v-model="name" class="w-full" />
        </UFormField>

        <div class="flex justify-end">
          <UButton type="submit" :loading="saving">
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UPageCard>

    <InboxSettingsForm :inbox="inbox" />
  </div>
</template>
