<script setup lang="ts">
defineProps<{
  contactId: string
}>()
</script>

<template>
  <NoteEditor :contact-id="contactId" />
</template>
