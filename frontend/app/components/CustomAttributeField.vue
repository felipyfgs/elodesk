<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { CustomAttributeDefinition } from '~/stores/customAttributes'

const props = defineProps<{
  definition: CustomAttributeDefinition
  value: unknown
  contactId: string
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const localValue = ref(props.value ?? '')
const saving = ref(false)

watch(() => props.value, (v) => {
  localValue.value = v ?? ''
})

async function save() {
  if (!auth.account?.id) return
  const key = props.definition.attributeKey
  const val = localValue.value

  saving.value = true
  try {
    if (val === '' || val === null || val === undefined) {
      await api(`/accounts/${auth.account.id}/contacts/${props.contactId}/custom_attributes`, {
        method: 'DELETE',
        body: { keys: [key] }
      })
    } else {
      const body: Record<string, unknown> = {}
      body[key] = val
      await api(`/accounts/${auth.account.id}/contacts/${props.contactId}/custom_attributes`, {
        method: 'POST',
        body
      })
    }
  } finally {
    saving.value = false
  }
}

const listOptions = computed(() => {
  if (props.definition.attributeDisplayType !== 'list' || !props.definition.attributeValues) return []
  try {
    const parsed = JSON.parse(props.definition.attributeValues)
    return Array.isArray(parsed) ? parsed.map(String) : []
  } catch {
    return []
  }
})

const inputType = computed(() => {
  const map: Record<string, string> = {
    number: 'number',
    currency: 'number',
    percent: 'number',
    date: 'date',
    link: 'url'
  }
  return map[props.definition.attributeDisplayType] ?? 'text'
})

function handleBlur() {
  if (localValue.value !== (props.value ?? '')) {
    save()
  }
}

function handleChange() {
  save()
}
</script>

<template>
  <UFormField :label="definition.attributeDisplayName">
    <UInput
      v-if="!['checkbox', 'list'].includes(definition.attributeDisplayType)"
      v-model="localValue"
      :type="inputType"
      class="w-full"
      :disabled="saving"
      @blur="handleBlur"
    />

    <UCheckbox
      v-else-if="definition.attributeDisplayType === 'checkbox'"
      :model-value="!!localValue"
      @update:model-value="(v: boolean) => { localValue = v; handleChange() }"
    />

    <USelect
      v-else-if="definition.attributeDisplayType === 'list'"
      :model-value="String(localValue ?? '')"
      :items="listOptions"
      class="w-full"
      :disabled="saving"
      @update:model-value="(v: string) => { localValue = v; handleChange() }"
    />
  </UFormField>
</template>
