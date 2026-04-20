<script setup lang="ts">
const _props = defineProps<{ hookId: number }>()
const { t } = useI18n()
const toast = useToast()

function fireTest() {
  toast.add({ title: t('settings.integrations.testSend'), description: 'stub — wire to backend trigger endpoint when implemented', color: 'info' })
}
</script>

<template>
  <UButton
    variant="outline"
    size="xs"
    icon="i-lucide-send"
    @click="fireTest"
  >
    {{ t('settings.integrations.testSend') }}
  </UButton>
</template>
