<script setup lang="ts">
import type { Contact } from '~/stores/contacts'

defineProps<{
  contact: Contact
  contactId: string
}>()

// Events timeline — placeholder until audit logs endpoint is available
const events = ref<{ id: string, type: string, description: string, createdAt: string }[]>([])
</script>

<template>
  <ContactsDetailContactEventsTimeline :events="events" />
</template>
