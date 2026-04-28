<script setup lang="ts">
import type { Inbox } from '~/stores/inboxes'

defineProps<{
  inbox: Inbox
}>()
</script>

<template>
  <InboxesSettingsWebhookSettings :inbox="inbox" />
</template>
