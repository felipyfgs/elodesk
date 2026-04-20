<script setup lang="ts">
import QRCode from 'qrcode'

const props = defineProps<{
  uri: string
  secret: string
}>()

const { t } = useI18n()

const qrDataUrl = ref('')

onMounted(async () => {
  qrDataUrl.value = await QRCode.toDataURL(props.uri, {
    width: 200,
    margin: 2,
    color: { dark: '#000000', light: '#ffffff' }
  })
})

async function copySecret() {
  await navigator.clipboard.writeText(props.secret)
}
</script>

<template>
  <div class="flex flex-col items-center gap-4">
    <div class="bg-white p-3 rounded-lg">
      <img
        v-if="qrDataUrl"
        :src="qrDataUrl"
        alt="QR Code"
        class="w-48 h-48"
      >
    </div>

    <p class="text-sm text-muted text-center">
      {{ t('auth.mfa.scanQr') }}
    </p>

    <div class="w-full">
      <p class="text-xs text-muted mb-1">
        {{ t('auth.mfa.setup') }}:
      </p>
      <div class="flex items-center gap-2">
        <code class="flex-1 text-xs bg-elevated p-2 rounded break-all select-all">
          {{ secret }}
        </code>
        <UButton
          variant="ghost"
          size="xs"
          icon="i-lucide-copy"
          @click="copySecret"
        />
      </div>
    </div>
  </div>
</template>
