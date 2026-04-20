<script setup lang="ts">
import { useCustomAttributesStore } from '~/stores/customAttributes'

defineProps<{
  contactId: string
  values: Record<string, unknown>
}>()

const { t } = useI18n()
const attrsStore = useCustomAttributesStore()
</script>

<template>
  <UPageCard v-if="attrsStore.contactDefinitions.length" variant="outline" :title="t('contactDetail.attributes')">
    <div class="space-y-3">
      <CustomAttributeField
        v-for="def in attrsStore.contactDefinitions"
        :key="def.id"
        :definition="def"
        :value="values?.[def.attributeKey]"
        :contact-id="contactId"
      />
    </div>
  </UPageCard>
</template>
