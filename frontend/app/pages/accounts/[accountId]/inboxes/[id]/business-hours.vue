<script setup lang="ts">
import type { Inbox } from '~/stores/inboxes'
import InboxBusinessHoursEditor from '~/components/inboxes/settings/InboxBusinessHoursEditor.vue'

defineProps<{
  inbox: Inbox
}>()
</script>

<template>
  <UPageCard :title="$t('inboxes.businessHours')" variant="subtle">
    <InboxBusinessHoursEditor :inbox="inbox" />
  </UPageCard>
</template>
