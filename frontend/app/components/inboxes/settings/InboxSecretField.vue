<script setup lang="ts">
const props = defineProps<{
  modelValue: string
}>()

const { t } = useI18n()
const revealed = ref(false)
const toast = useToast()

function toggle() {
  revealed.value = !revealed.value
}

async function copy() {
  try {
    await navigator.clipboard.writeText(props.modelValue)
    toast.add({ title: t('common.copied'), color: 'success' })
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  }
}
</script>

<template>
  <div class="flex items-center gap-1">
    <UInput
      :model-value="revealed ? modelValue : '••••••••••••••••'"
      readonly
      class="font-mono text-xs"
    />
    <UTooltip :text="revealed ? t('inboxes.secret.hide') : t('inboxes.secret.reveal')">
      <UButton
        :icon="revealed ? 'i-lucide-eye-off' : 'i-lucide-eye'"
        variant="ghost"
        color="neutral"
        size="sm"
        @click="toggle"
      />
    </UTooltip>
    <UTooltip :text="t('inboxes.secret.copy')">
      <UButton
        icon="i-lucide-copy"
        variant="ghost"
        color="neutral"
        size="sm"
        @click="copy"
      />
    </UTooltip>
  </div>
</template>
