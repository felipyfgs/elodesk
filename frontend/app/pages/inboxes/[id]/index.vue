<script setup lang="ts">
import type { Inbox } from '~/stores/inboxes'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
</script>

<template>
  <UPageCard :title="t('inboxes.general')" variant="subtle">
    <dl class="text-sm flex flex-col gap-3 mt-3">
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          {{ t('inboxes.wizards.name') }}
        </dt>
        <dd class="font-medium">
          {{ props.inbox.name }}
        </dd>
      </div>
      <USeparator />
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          {{ t('inboxes.channels._') }}
        </dt>
        <dd>{{ t(`inboxes.channels.${props.inbox.channelType}`) }}</dd>
      </div>
      <USeparator />
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          ID
        </dt>
        <dd class="font-mono text-xs">
          {{ props.inbox.id }}
        </dd>
      </div>
      <USeparator />
      <div class="flex gap-3">
        <dt class="text-muted w-40 shrink-0">
          {{ t('common.createdAt', {}, { locale: 'pt-BR' }) }}
        </dt>
        <dd>{{ new Date(props.inbox.createdAt).toLocaleDateString() }}</dd>
      </div>
    </dl>
  </UPageCard>
</template>
