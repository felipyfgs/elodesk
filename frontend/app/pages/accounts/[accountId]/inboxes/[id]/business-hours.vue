<script setup lang="ts">
import type { Inbox } from '~/stores/inboxes'

defineProps<{
  inbox: Inbox
}>()
</script>

<template>
  <UPageCard :title="$t('inboxes.businessHours')" variant="subtle">
    <InboxBusinessHoursEditor :inbox="inbox" />
  </UPageCard>
</template>
