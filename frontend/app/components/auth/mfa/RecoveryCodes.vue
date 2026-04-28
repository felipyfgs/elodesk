<script setup lang="ts">
const props = defineProps<{
  codes: string[]
}>()

const emit = defineEmits<{
  done: []
}>()

const { t } = useI18n()

async function copyAll() {
  await navigator.clipboard.writeText(props.codes.join('\n'))
}

function downloadTxt() {
  const blob = new Blob([props.codes.join('\n')], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'elodesk-recovery-codes.txt'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <p class="text-sm text-muted text-center">
      {{ t('auth.mfa.recoveryCodes') }}
    </p>

    <div class="grid grid-cols-2 gap-2">
      <code
        v-for="(code, i) in codes"
        :key="i"
        class="text-xs bg-elevated p-2 rounded text-center select-all font-mono"
      >
        {{ code }}
      </code>
    </div>

    <div class="flex gap-2 justify-center">
      <UButton
        variant="outline"
        size="sm"
        icon="i-lucide-copy"
        @click="copyAll"
      >
        {{ t('common.copied') }}
      </UButton>
      <UButton
        variant="outline"
        size="sm"
        icon="i-lucide-download"
        @click="downloadTxt"
      >
        {{ t('auth.mfa.downloadCodes') }}
      </UButton>
    </div>

    <UButton block @click="emit('done')">
      {{ t('common.save') }}
    </UButton>
  </div>
</template>
