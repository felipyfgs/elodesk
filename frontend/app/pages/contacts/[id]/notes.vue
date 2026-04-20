<script setup lang="ts">
import type { Contact } from '~/stores/contacts'

defineProps<{
  contact: Contact
  contactId: string
}>()
</script>

<template>
  <NoteEditor :contact-id="contactId" />
</template>
