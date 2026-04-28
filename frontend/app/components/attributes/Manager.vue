<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { ConfirmModal } from '#components'
import { useAuthStore } from '~/stores/auth'
import { useCustomAttributesStore, type CustomAttributeDefinition } from '~/stores/customAttributes'
import { customAttributeSchema, type CustomAttributeForm } from '~/schemas/customAttribute'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useCustomAttributesStore()
const confirm = useOverlay().create(ConfirmModal)

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<CustomAttributeDefinition | null>(null)
const saved = ref(false)

const displayTypes = ['text', 'number', 'currency', 'percent', 'link', 'date', 'list', 'checkbox']

const form = reactive<CustomAttributeForm>({
  attribute_key: '',
  attribute_display_name: '',
  attribute_display_type: 'text',
  attribute_model: 'contact',
  attribute_values: null,
  attribute_description: null,
  regex_pattern: null,
  default_value: null
})

const loading = ref(false)

function resetForm() {
  form.attribute_key = ''
  form.attribute_display_name = ''
  form.attribute_display_type = 'text'
  form.attribute_model = 'contact'
  form.attribute_values = null
  form.attribute_description = null
  form.regex_pattern = null
  form.default_value = null
  editing.value = null
}

function openCreate() {
  resetForm()
  open.value = true
}

function openEdit(def: CustomAttributeDefinition) {
  editing.value = def
  form.attribute_key = def.attributeKey
  form.attribute_display_name = def.attributeDisplayName
  form.attribute_display_type = def.attributeDisplayType
  form.attribute_model = def.attributeModel
  form.attribute_values = def.attributeValues
  form.attribute_description = def.attributeDescription
  form.regex_pattern = def.regexPattern
  form.default_value = def.defaultValue
  open.value = true
}

async function submit(event: FormSubmitEvent<CustomAttributeForm>) {
  loading.value = true
  try {
    const body = {
      ...event.data,
      attribute_values: event.data.attribute_display_type === 'list'
        ? (typeof event.data.attribute_values === 'string' ? event.data.attribute_values : null)
        : null
    }
    if (!auth.account?.id) return
    const base = `/accounts/${auth.account.id}/custom_attribute_definitions`
    if (editing.value) {
      const updated = await api<CustomAttributeDefinition>(`${base}/${editing.value.id}`, { method: 'PATCH', body })
      store.upsert(updated)
    } else {
      const created = await api<CustomAttributeDefinition>(base, { method: 'POST', body })
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

function openDelete(def: CustomAttributeDefinition) {
  confirm.open({
    title: t('common.delete'),
    description: t('customAttributes.deleteConfirm'),
    confirmLabel: t('common.delete'),
    itemName: def.attributeDisplayName
  }).then(async (ok) => {
    if (!ok) return
    if (!auth.account?.id) return
    await api(`/accounts/${auth.account.id}/custom_attribute_definitions/${def.id}`, { method: 'DELETE' })
    store.remove(def.id, def.attributeModel)
  })
}

function allDefinitions(): CustomAttributeDefinition[] {
  return [...store.contactDefinitions, ...store.conversationDefinitions]
}

async function fetchAttributes() {
  if (!auth.account?.id) return
  const list = await api<CustomAttributeDefinition[]>(`/accounts/${auth.account.id}/custom_attribute_definitions`)
  store.setAll(list)
}

onMounted(fetchAttributes)
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

    <UPageCard :title="t('customAttributes.title')" variant="subtle">
      <template #header>
        <UButton @click="openCreate">
          {{ t('customAttributes.create') }}
        </UButton>
      </template>

      <p v-if="!allDefinitions().length" class="text-sm text-muted">
        {{ t('customAttributes.empty') }}
      </p>

      <div v-else class="flex flex-col gap-2 mt-2">
        <div
          v-for="def in allDefinitions()"
          :key="def.id"
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-default"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ def.attributeDisplayName }}</span>
              <UBadge variant="subtle">
                {{ def.attributeKey }}
              </UBadge>
              <UBadge variant="subtle">
                {{ t(`customAttributes.types.${def.attributeDisplayType}`) }}
              </UBadge>
              <UBadge>{{ def.attributeModel }}</UBadge>
            </div>
            <div v-if="def.attributeDescription" class="text-sm text-muted truncate mt-1">
              {{ def.attributeDescription }}
            </div>
          </div>
          <div class="flex gap-1 shrink-0">
            <UButton size="xs" variant="ghost" @click="openEdit(def)">
              {{ t('common.edit') }}
            </UButton>
            <UButton
              size="xs"
              color="error"
              variant="ghost"
              @click="openDelete(def)"
            >
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('customAttributes.edit') : t('customAttributes.create')">
      <UForm
        :schema="customAttributeSchema"
        :state="form"
        class="flex flex-col gap-4"
        @submit="submit"
      >
        <UFormField :label="t('customAttributes.key')" name="attribute_key">
          <UInput v-model="form.attribute_key" class="w-full" placeholder="ex: department" />
        </UFormField>

        <UFormField :label="t('customAttributes.displayName')" name="attribute_display_name">
          <UInput v-model="form.attribute_display_name" class="w-full" />
        </UFormField>

        <UFormField :label="t('customAttributes.displayType')" name="attribute_display_type">
          <USelect
            v-model="form.attribute_display_type"
            :items="displayTypes"
            class="w-full"
          />
        </UFormField>

        <UFormField :label="t('customAttributes.model')" name="attribute_model">
          <USelect
            v-model="form.attribute_model"
            :items="['contact', 'conversation']"
            class="w-full"
          />
        </UFormField>

        <UFormField v-if="form.attribute_display_type === 'list'" :label="t('customAttributes.values')" name="attribute_values">
          <UTextarea
            v-model="form.attribute_values!"
            class="w-full"
            :placeholder="t('customAttributes.valuesPlaceholder')"
          />
        </UFormField>

        <UFormField :label="t('customAttributes.description')" name="attribute_description">
          <UTextarea v-model="form.attribute_description!" class="w-full" />
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
