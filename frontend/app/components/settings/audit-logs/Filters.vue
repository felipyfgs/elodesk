<script setup lang="ts">
const props = defineProps<{ from?: string, to?: string, action?: string, entityType?: string }>()
const emit = defineEmits<{
  'update:from': [value: string]
  'update:to': [value: string]
  'update:action': [value: string]
  'update:entityType': [value: string]
}>()

const { t } = useI18n()

const fromModel = computed({ get: () => props.from ?? '', set: v => emit('update:from', v) })
const toModel = computed({ get: () => props.to ?? '', set: v => emit('update:to', v) })
const actionModel = computed({ get: () => props.action ?? '', set: v => emit('update:action', v) })
const entityModel = computed({ get: () => props.entityType ?? '', set: v => emit('update:entityType', v) })
</script>

<template>
  <div class="space-y-3">
    <UFormField :label="t('settings.auditLogs.date')">
      <UInput v-model="fromModel" type="date" />
      <UInput v-model="toModel" type="date" />
    </UFormField>
    <UFormField :label="t('settings.auditLogs.action')">
      <UInput v-model="actionModel" />
    </UFormField>
    <UFormField :label="t('settings.auditLogs.entity')">
      <UInput v-model="entityModel" />
    </UFormField>
  </div>
</template>
