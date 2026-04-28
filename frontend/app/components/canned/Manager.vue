<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { ConfirmModal } from '#components'
import { useAuthStore } from '~/stores/auth'
import { useCannedResponsesStore, type CannedResponse } from '~/stores/cannedResponses'
import { cannedResponseSchema, type CannedResponseForm } from '~/schemas/cannedResponse'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useCannedResponsesStore()
const confirm = useOverlay().create(ConfirmModal)

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<CannedResponse | null>(null)
const saved = ref(false)

const form = reactive<CannedResponseForm>({
  short_code: '',
  content: ''
})

const loading = ref(false)

function resetForm() {
  form.short_code = ''
  form.content = ''
  editing.value = null
}

function openCreate() {
  resetForm()
  open.value = true
}

function openEdit(item: CannedResponse) {
  editing.value = item
  form.short_code = item.shortCode
  form.content = item.content
  open.value = true
}

async function submit(event: FormSubmitEvent<CannedResponseForm>) {
  loading.value = true
  try {
    if (editing.value) {
      const updated = await api<CannedResponse>(`/accounts/${auth.account?.id}/canned_responses/${editing.value.id}`, { method: 'PATCH', body: event.data })
      store.upsert(updated)
    } else {
      const created = await api<CannedResponse>(`/accounts/${auth.account?.id}/canned_responses`, { method: 'POST', body: event.data })
      store.upsert(created)
    }
    saved.value = true
    open.value = false
    resetForm()
    setTimeout(() => {
      saved.value = false
    }, 2000)
  } finally {
    loading.value = false
  }
}

function openDelete(item: CannedResponse) {
  confirm.open({
    title: t('common.delete'),
    description: t('cannedResponses.deleteConfirm'),
    confirmLabel: t('common.delete'),
    itemName: item.shortCode
  }).then(async (ok) => {
    if (!ok) return
    await api(`/accounts/${auth.account?.id}/canned_responses/${item.id}`, { method: 'DELETE' })
    store.remove(item.id)
  })
}

async function fetchItems() {
  const list = await api<CannedResponse[]>(`/accounts/${auth.account?.id}/canned_responses`)
  store.setAll(list)
}

onMounted(fetchItems)
</script>

<template>
  <div v-if="!isAdmin" class="text-center text-muted py-12">
    {{ t('settings.accessDenied') }}
  </div>

  <template v-else>
    <UAlert
      v-if="saved"
      class="mb-4"
      color="success"
      variant="subtle"
      icon="i-lucide-check-circle"
      :title="t('common.success')"
    />

    <UPageCard :title="t('cannedResponses.title')" variant="subtle">
      <template #header>
        <UButton @click="openCreate">
          {{ t('cannedResponses.create') }}
        </UButton>
      </template>

      <p v-if="!store.list.length" class="text-sm text-muted">
        {{ t('cannedResponses.empty') }}
      </p>

      <div v-else class="flex flex-col gap-2 mt-2">
        <div
          v-for="item in store.list"
          :key="item.id"
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-default"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <UBadge variant="subtle">
                {{ item.shortCode }}
              </UBadge>
            </div>
            <div class="text-sm text-muted truncate mt-1">
              {{ item.content.slice(0, 120) }}{{ item.content.length > 120 ? '...' : '' }}
            </div>
          </div>
          <div class="flex gap-1 shrink-0">
            <UButton size="xs" variant="ghost" @click="openEdit(item)">
              {{ t('common.edit') }}
            </UButton>
            <UButton
              size="xs"
              color="error"
              variant="ghost"
              @click="openDelete(item)"
            >
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('cannedResponses.edit') : t('cannedResponses.create')">
      <UForm
        :schema="cannedResponseSchema"
        :state="form"
        class="flex flex-col gap-4"
        @submit="submit"
      >
        <UFormField :label="t('cannedResponses.shortCode')" name="short_code">
          <UInput v-model="form.short_code" class="w-full" placeholder="ex: greeting" />
        </UFormField>

        <UFormField :label="t('cannedResponses.content')" name="content">
          <UTextarea v-model="form.content" class="w-full" :rows="6" />
        </UFormField>

        <div class="flex justify-end gap-2">
          <UButton type="button" variant="ghost" @click="open = false">
            {{ t('common.cancel') }}
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UModal>
  </template>
</template>
