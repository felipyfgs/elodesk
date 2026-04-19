<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore, type Label } from '~/stores/labels'
import { labelSchema, type LabelForm } from '~/schemas/label'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useLabelsStore()

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<Label | null>(null)
const saved = ref(false)
const errors = ref<Record<string, string>>({})

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
  errors.value = {}
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
  errors.value = {}
  open.value = true
}

async function submit() {
  const result = labelSchema.safeParse(form)
  if (!result.success) {
    errors.value = Object.fromEntries(result.error.issues.map(i => [i.path.join('.'), i.message]))
    return
  }
  errors.value = {}
  loading.value = true
  try {
    if (editing.value) {
      const updated = await api<Label>(`/labels/${editing.value.id}`, { method: 'PUT', body: result.data })
      store.upsert(updated)
    } else {
      const created = await api<Label>('/labels', { method: 'POST', body: result.data })
      store.upsert(created)
    }
    saved.value = true
    open.value = false
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    loading.value = false
  }
}

async function remove(label: Label) {
  if (!confirm(t('labels.deleteConfirm'))) return
  await api(`/labels/${label.id}`, { method: 'DELETE' })
  store.remove(label.id)
}

async function fetchLabels() {
  const list = await api<Label[]>('/labels')
  store.setAll(list)
}

onMounted(fetchLabels)
</script>

<template>
  <div v-if="!isAdmin" class="text-center text-muted py-12">
    {{ t('settings.accessDenied') }}
  </div>

  <template v-else>
    <div v-if="saved" class="mb-4 text-sm text-green-600">
      {{ t('common.success') }}
    </div>

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
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-[var(--ui-border)]"
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
            <UButton size="xs" color="red" variant="ghost" @click="remove(label)">
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('labels.edit') : t('labels.create')">
      <form class="flex flex-col gap-4" @submit.prevent="submit">
        <UFormField :label="t('labels.name')" :error="errors.title">
          <UInput v-model="form.title" class="w-full" />
        </UFormField>

        <UFormField :label="t('labels.color')" :error="errors.color">
          <div class="flex gap-2 items-center w-full">
            <input v-model="form.color" type="color" class="h-9 w-12 cursor-pointer rounded border border-[var(--ui-border)]" />
            <UInput v-model="form.color" class="flex-1" />
          </div>
        </UFormField>

        <UFormField :label="t('labels.description')" :error="errors.description">
          <UTextarea v-model="form.description!" class="w-full" />
        </UFormField>

        <UFormField>
          <UCheckbox v-model="form.show_on_sidebar" :label="t('labels.showOnSidebar')" />
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
