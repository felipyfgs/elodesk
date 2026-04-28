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
const errorHandler = useErrorHandler()

const open = ref(false)
const editing = ref<Label | null>(null)
const fetching = ref(false)
const loading = ref(false)
const formRef = useTemplateRef('formRef')

const initialState = (): LabelForm => ({
  title: '',
  color: '#1f93ff',
  description: null,
  show_on_sidebar: false
})

const form = reactive<LabelForm>(initialState())

function resetForm() {
  Object.assign(form, initialState())
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

watch(open, (value) => {
  if (!value) resetForm()
})

async function submit(event: FormSubmitEvent<LabelForm>) {
  loading.value = true
  try {
    const body = {
      ...event.data,
      description: event.data.description?.trim() ? event.data.description : null
    }
    const base = `/accounts/${auth.account?.id}/labels`
    if (editing.value) {
      const updated = await api<Label>(`${base}/${editing.value.id}`, { method: 'PATCH', body })
      store.upsert(updated)
      errorHandler.success(t('common.success'), t('labels.updateSuccess'))
    } else {
      const created = await api<Label>(base, { method: 'POST', body })
      store.upsert(created)
      errorHandler.success(t('common.success'), t('labels.createSuccess'))
    }
    open.value = false
  } catch (error) {
    errorHandler.handle(error, {
      title: editing.value ? t('labels.updateFailed') : t('labels.createFailed')
    })
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
    try {
      await api(`/accounts/${auth.account?.id}/labels/${label.id}`, { method: 'DELETE' })
      store.remove(label.id)
      errorHandler.success(t('common.success'), t('labels.deleteSuccess'))
    } catch (error) {
      errorHandler.handle(error, { title: t('labels.deleteFailed') })
    }
  })
}

async function fetchLabels() {
  fetching.value = true
  try {
    const list = await api<Label[]>(`/accounts/${auth.account?.id}/labels`)
    store.setAll(list)
  } catch (error) {
    errorHandler.handle(error, {
      title: t('labels.fetchFailed'),
      onRetry: fetchLabels
    })
  } finally {
    fetching.value = false
  }
}

onMounted(fetchLabels)
</script>

<template>
  <UPageCard
    :title="t('labels.title')"
    :description="t('labels.subtitle')"
    variant="subtle"
  >
    <template #header>
      <UButton
        icon="i-lucide-plus"
        @click="openCreate"
      >
        {{ t('labels.create') }}
      </UButton>
    </template>

    <div v-if="fetching" class="flex flex-col gap-2 mt-2">
      <USkeleton v-for="i in 3" :key="i" class="h-12 w-full" />
    </div>

    <div
      v-else-if="!store.list.length"
      class="flex flex-col items-center text-center gap-2 py-10"
    >
      <UIcon name="i-lucide-tag" class="size-10 text-muted" />
      <p class="text-sm text-muted">
        {{ t('labels.empty') }}
      </p>
      <UButton variant="ghost" icon="i-lucide-plus" @click="openCreate">
        {{ t('labels.create') }}
      </UButton>
    </div>

    <div v-else class="flex flex-col gap-2 mt-2">
      <div
        v-for="label in store.list"
        :key="label.id"
        class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-default bg-default hover:bg-elevated transition-colors"
      >
        <div class="flex items-center gap-3 min-w-0">
          <UBadge
            :style="{ backgroundColor: label.color, color: '#fff' }"
            class="shrink-0"
          >
            {{ label.title }}
          </UBadge>
          <span v-if="label.description" class="text-sm text-muted truncate">
            {{ label.description }}
          </span>
          <UIcon
            v-if="label.showOnSidebar"
            name="i-lucide-pin"
            class="size-3.5 text-muted shrink-0"
            :title="t('labels.showOnSidebar')"
          />
        </div>
        <div class="flex gap-1 shrink-0">
          <UButton
            size="xs"
            variant="ghost"
            icon="i-lucide-pencil"
            :aria-label="t('common.edit')"
            @click="openEdit(label)"
          />
          <UButton
            size="xs"
            color="error"
            variant="ghost"
            icon="i-lucide-trash-2"
            :aria-label="t('common.delete')"
            @click="openDelete(label)"
          />
        </div>
      </div>
    </div>

    <UModal
      v-model:open="open"
      :title="editing ? t('labels.edit') : t('labels.create')"
      :ui="{ content: 'sm:max-w-md' }"
    >
      <template #body>
        <UForm
          ref="formRef"
          :schema="labelSchema"
          :state="form"
          class="flex flex-col gap-4"
          @submit="submit"
        >
          <UFormField :label="t('labels.name')" name="title" required>
            <UInput v-model="form.title" autofocus class="w-full" />
          </UFormField>

          <UFormField :label="t('labels.color')" name="color">
            <div class="flex gap-2 items-center w-full">
              <UPopover :content="{ side: 'bottom', align: 'start' }">
                <UButton
                  type="button"
                  color="neutral"
                  variant="outline"
                  class="shrink-0"
                >
                  <span
                    class="size-4 rounded-full ring ring-default"
                    :style="{ backgroundColor: form.color }"
                  />
                </UButton>

                <template #content>
                  <UColorPicker v-model="form.color" class="p-2" />
                </template>
              </UPopover>
              <UInput v-model="form.color" class="flex-1" />
            </div>
          </UFormField>

          <UFormField :label="t('labels.description')" name="description">
            <UTextarea
              :model-value="form.description ?? ''"
              :rows="3"
              autoresize
              class="w-full"
              @update:model-value="value => form.description = value ? String(value) : null"
            />
          </UFormField>

          <UFormField name="show_on_sidebar">
            <UCheckbox v-model="form.show_on_sidebar" :label="t('labels.showOnSidebar')" />
          </UFormField>
        </UForm>
      </template>

      <template #footer>
        <div class="flex justify-end items-center gap-2 w-full">
          <UButton
            color="neutral"
            variant="ghost"
            :disabled="loading"
            @click="open = false"
          >
            {{ t('common.cancel') }}
          </UButton>
          <UButton :loading="loading" @click="formRef?.submit()">
            {{ t('common.save') }}
          </UButton>
        </div>
      </template>
    </UModal>
  </UPageCard>
</template>
