<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const auth = useAuthStore()
</script>

<template>
  <UPageCard
    :title="t('settings.general.title')"
    :description="auth.user?.email"
    variant="subtle"
  >
    <dl class="text-sm flex flex-col gap-3 mt-3">
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          {{ t('settings.general.name') }}
        </dt>
        <dd class="font-medium">
          {{ auth.user?.name ?? '—' }}
        </dd>
      </div>
      <USeparator />
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          {{ t('settings.general.email') }}
        </dt>
        <dd>{{ auth.user?.email ?? '—' }}</dd>
      </div>
      <USeparator />
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          {{ t('settings.general.organization') }}
        </dt>
        <dd>{{ auth.account?.name ?? '—' }}</dd>
      </div>
    </dl>
  </UPageCard>
</template>
