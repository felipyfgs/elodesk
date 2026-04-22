<script setup lang="ts">
import type { Contact } from '~/stores/contacts'
import { useCustomAttributesStore } from '~/stores/customAttributes'

const props = defineProps<{
  contact: Contact
}>()

const { t } = useI18n()
const attrsStore = useCustomAttributesStore()

const values = computed<Record<string, unknown>>(() => {
  if (!props.contact.additionalAttributes) return {}
  try {
    return JSON.parse(props.contact.additionalAttributes) as Record<string, unknown>
  } catch {
    return {}
  }
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-medium text-highlighted">
      {{ t('contactDetail.sidebar.attributes') }}
    </h3>

    <p v-if="!attrsStore.contactDefinitions.length" class="text-sm text-muted">
      {{ t('contactDetail.sidebar.noAttributes') }}
    </p>

    <div v-else class="flex flex-col gap-3">
      <CustomAttributeField
        v-for="def in attrsStore.contactDefinitions"
        :key="def.id"
        :definition="def"
        :value="values?.[def.attributeKey]"
        :contact-id="String(contact.id)"
      />
    </div>
  </div>
</template>
