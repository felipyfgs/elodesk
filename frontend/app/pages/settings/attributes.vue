<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useCustomAttributesStore, type CustomAttributeDefinition } from '~/stores/customAttributes'
import { customAttributeSchema, type CustomAttributeForm } from '~/schemas/customAttribute'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useCustomAttributesStore()

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<CustomAttributeDefinition | null>(null)
const saved = ref(false)
const errors = ref<Record<string, string>>({})

const displayTypes = ['text', 'number', 'currency', 'percent', 'link', 'date', 'list', 'checkbox'] as const

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
  errors.value = {}
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
  errors.value = {}
  open.value = true
}

async function submit() {
  const result = customAttributeSchema.safeParse(form)
  if (!result.success) {
    errors.value = Object.fromEntries(result.error.issues.map(i => [i.path.join('.'), i.message]))
    return
  }
  errors.value = {}
  loading.value = true
  try {
    const body = {
      ...result.data,
      attribute_values: result.data.attribute_display_type === 'list'
        ? (typeof result.data.attribute_values === 'string' ? result.data.attribute_values : null)
        : null
    }
    if (editing.value) {
      const updated = await api<CustomAttributeDefinition>(`/custom-attributes/${editing.value.id}`, { method: 'PUT', body })
      store.upsert(updated)
    } else {
      const created = await api<CustomAttributeDefinition>('/custom-attributes', { method: 'POST', body })
      store.upsert(created)
    }
    saved.value = true
    open.value = false
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    loading.value = false
  }
}

async function remove(def: CustomAttributeDefinition) {
  if (!confirm(t('customAttributes.deleteConfirm'))) return
  await api(`/custom-attributes/${def.id}`, { method: 'DELETE' })
  store.remove(def.id, def.attributeModel)
}

function allDefinitions(): CustomAttributeDefinition[] {
  return [...store.contactDefinitions, ...store.conversationDefinitions]
}

async function fetchAttributes() {
  const list = await api<CustomAttributeDefinition[]>('/custom-attributes')
  store.setAll(list)
}

onMounted(fetchAttributes)
</script>

<template>
  <div v-if="!isAdmin" class="text-center text-muted py-12">
    {{ t('settings.accessDenied') }}
  </div>

  <template v-else>
    <div v-if="saved" class="mb-4 text-sm text-green-600">
      {{ t('common.success') }}
    </div>

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
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-[var(--ui-border)]"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ def.attributeDisplayName }}</span>
              <UBadge variant="subtle">{{ def.attributeKey }}</UBadge>
              <UBadge variant="subtle">{{ t(`customAttributes.types.${def.attributeDisplayType}`) }}</UBadge>
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
            <UButton size="xs" color="red" variant="ghost" @click="remove(def)">
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('customAttributes.edit') : t('customAttributes.create')">
      <form class="flex flex-col gap-4" @submit.prevent="submit">
        <UFormField :label="t('customAttributes.key')" :error="errors.attribute_key">
          <UInput v-model="form.attribute_key" class="w-full" placeholder="ex: department" />
        </UFormField>

        <UFormField :label="t('customAttributes.displayName')" :error="errors.attribute_display_name">
          <UInput v-model="form.attribute_display_name" class="w-full" />
        </UFormField>

        <UFormField :label="t('customAttributes.displayType')" :error="errors.attribute_display_type">
          <USelect
            v-model="form.attribute_display_type"
            :options="displayTypes"
            class="w-full"
          >
            <template #option="{ option }">
              {{ t(`customAttributes.types.${option}`) }}
            </template>
            <template #selected="{ modelValue }">
              {{ t(`customAttributes.types.${modelValue}`) }}
            </template>
          </USelect>
        </UFormField>

        <UFormField :label="t('customAttributes.model')" :error="errors.attribute_model">
          <USelect
            v-model="form.attribute_model"
            :options="['contact', 'conversation']"
            class="w-full"
          />
        </UFormField>

        <UFormField v-if="form.attribute_display_type === 'list'" :label="t('customAttributes.values')" :error="errors.attribute_values">
          <UTextarea
            v-model="form.attribute_values!"
            class="w-full"
            :placeholder="t('customAttributes.valuesPlaceholder')"
          />
        </UFormField>

        <UFormField :label="t('customAttributes.description')" :error="errors.attribute_description">
          <UTextarea v-model="form.attribute_description!" class="w-full" />
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
