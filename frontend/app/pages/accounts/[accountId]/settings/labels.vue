<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { ConfirmModal } from '#components'
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore, type Label } from '~/stores/labels'
import { labelSchema, type LabelForm } from '~/schemas/label'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useLabelsStore()
const confirm = useOverlay().create(ConfirmModal)

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<Label | null>(null)
const saved = ref(false)

const form = reactive<LabelForm>({
  title: '',
  color: '#1f93ff',
  description: null,
  show_on_sidebar: false
})

const loading = ref(false)

function resetForm() {
  form.title = ''
  form.color = '#1f93ff'
  form.description = null
  form.show_on_sidebar = false
  editing.value = null
}

function openCreate() {
  resetForm()
  open.value = true
}

function openEdit(label: Label) {
  editing.value = label
  form.title = label.title
  form.color = label.color
  form.description = label.description
  form.show_on_sidebar = label.showOnSidebar
  open.value = true
}

async function submit(event: FormSubmitEvent<LabelForm>) {
  loading.value = true
  try {
    if (editing.value) {
      const updated = await api<Label>(`/accounts/${auth.account?.id}/labels/${editing.value.id}`, { method: 'PATCH', body: event.data })
      store.upsert(updated)
    } else {
      const created = await api<Label>(`/accounts/${auth.account?.id}/labels`, { method: 'POST', body: event.data })
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

function openDelete(label: Label) {
  confirm.open({
    title: t('common.delete'),
    description: t('labels.deleteConfirm'),
    confirmLabel: t('common.delete'),
    itemName: label.title
  }).then(async (ok) => {
    if (!ok) return
    await api(`/accounts/${auth.account?.id}/labels/${label.id}`, { method: 'DELETE' })
    store.remove(label.id)
  })
}

async function fetchLabels() {
  const list = await api<Label[]>(`/accounts/${auth.account?.id}/labels`)
  store.setAll(list)
}

onMounted(fetchLabels)
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

    <UPageCard :title="t('labels.title')" variant="subtle">
      <template #header>
        <UButton @click="openCreate">
          {{ t('labels.create') }}
        </UButton>
      </template>

      <p v-if="!store.list.length" class="text-sm text-muted">
        {{ t('labels.empty') }}
      </p>

      <div v-else class="flex flex-col gap-2 mt-2">
        <div
          v-for="label in store.list"
          :key="label.id"
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-default"
        >
          <div class="flex items-center gap-2 min-w-0">
            <UBadge :style="{ backgroundColor: label.color, color: '#fff' }">
              {{ label.title }}
            </UBadge>
            <span v-if="label.description" class="text-sm text-muted truncate">
              {{ label.description }}
            </span>
          </div>
          <div class="flex gap-1 shrink-0">
            <UButton size="xs" variant="ghost" @click="openEdit(label)">
              {{ t('common.edit') }}
            </UButton>
            <UButton
              size="xs"
              color="error"
              variant="ghost"
              @click="openDelete(label)"
            >
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('labels.edit') : t('labels.create')">
      <UForm
        :schema="labelSchema"
        :state="form"
        class="flex flex-col gap-4"
        @submit="submit"
      >
        <UFormField :label="t('labels.name')" name="title">
          <UInput v-model="form.title" class="w-full" />
        </UFormField>

        <UFormField :label="t('labels.color')" name="color">
          <div class="flex gap-2 items-center w-full">
            <UColorPicker v-model="form.color" size="sm" />
            <UInput v-model="form.color" class="flex-1" />
          </div>
        </UFormField>

        <UFormField :label="t('labels.description')" name="description">
          <UTextarea v-model="form.description!" class="w-full" />
        </UFormField>

        <UFormField name="show_on_sidebar">
          <UCheckbox v-model="form.show_on_sidebar" :label="t('labels.showOnSidebar')" />
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
