<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useCannedResponsesStore, type CannedResponse } from '~/stores/cannedResponses'
import { cannedResponseSchema, type CannedResponseForm } from '~/schemas/cannedResponse'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useCannedResponsesStore()

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<CannedResponse | null>(null)
const saved = ref(false)
const errors = ref<Record<string, string>>({})

const form = reactive<CannedResponseForm>({
  short_code: '',
  content: ''
})

const loading = ref(false)

function resetForm() {
  form.short_code = ''
  form.content = ''
  errors.value = {}
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
  errors.value = {}
  open.value = true
}

async function submit() {
  const result = cannedResponseSchema.safeParse(form)
  if (!result.success) {
    errors.value = Object.fromEntries(result.error.issues.map(i => [i.path.join('.'), i.message]))
    return
  }
  errors.value = {}
  loading.value = true
  try {
    if (editing.value) {
      const updated = await api<CannedResponse>(`/canned-responses/${editing.value.id}`, { method: 'PUT', body: result.data })
      store.upsert(updated)
    } else {
      const created = await api<CannedResponse>('/canned-responses', { method: 'POST', body: result.data })
      store.upsert(created)
    }
    saved.value = true
    open.value = false
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    loading.value = false
  }
}

async function remove(item: CannedResponse) {
  if (!confirm(t('cannedResponses.deleteConfirm'))) return
  await api(`/canned-responses/${item.id}`, { method: 'DELETE' })
  store.remove(item.id)
}

async function fetchItems() {
  const list = await api<CannedResponse[]>('/canned-responses')
  store.setAll(list)
}

onMounted(fetchItems)
</script>

<template>
  <div v-if="!isAdmin" class="text-center text-muted py-12">
    {{ t('settings.accessDenied') }}
  </div>

  <template v-else>
    <div v-if="saved" class="mb-4 text-sm text-green-600">
      {{ t('common.success') }}
    </div>

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
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-[var(--ui-border)]"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <UBadge variant="subtle">{{ item.shortCode }}</UBadge>
            </div>
            <div class="text-sm text-muted truncate mt-1">
              {{ item.content.slice(0, 120) }}{{ item.content.length > 120 ? '...' : '' }}
            </div>
          </div>
          <div class="flex gap-1 shrink-0">
            <UButton size="xs" variant="ghost" @click="openEdit(item)">
              {{ t('common.edit') }}
            </UButton>
            <UButton size="xs" color="red" variant="ghost" @click="remove(item)">
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('cannedResponses.edit') : t('cannedResponses.create')">
      <form class="flex flex-col gap-4" @submit.prevent="submit">
        <UFormField :label="t('cannedResponses.shortCode')" :error="errors.short_code">
          <UInput v-model="form.short_code" class="w-full" placeholder="ex: greeting" />
        </UFormField>

        <UFormField :label="t('cannedResponses.content')" :error="errors.content">
          <UTextarea v-model="form.content" class="w-full" :rows="6" />
        </UFormField>

        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="open = false">
            {{ t('common.cancel') }}
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ t('common.save') }}
          </UButton>
        </div>
      </form>
    </UModal>
  </template>
</template>
