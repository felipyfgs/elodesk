<script setup lang="ts">
defineProps<{
  enabled: boolean
}>()

const emit = defineEmits<{
  setup: []
  disable: []
}>()

const { t } = useI18n()
</script>

<template>
  <UPageCard>
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div
          class="w-3 h-3 rounded-full"
          :class="enabled ? 'bg-success' : 'bg-muted'"
        />
        <div>
          <p class="text-sm font-medium">
            {{ t('auth.mfa.title') }}
          </p>
          <p class="text-xs text-muted">
            {{ enabled ? t('auth.mfa.enabled') : t('auth.mfa.disabled') }}
          </p>
        </div>
      </div>

      <UButton
        v-if="enabled"
        variant="outline"
        color="error"
        size="sm"
        @click="emit('disable')"
      >
        {{ t('auth.mfa.disable') }}
      </UButton>

      <UButton
        v-else
        variant="outline"
        size="sm"
        @click="emit('setup')"
      >
        {{ t('auth.mfa.setup') }}
      </UButton>
    </div>
  </UPageCard>
</template>
