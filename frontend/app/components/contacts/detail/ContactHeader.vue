<script setup lang="ts">
import type { Contact } from '~/stores/contacts'

const props = defineProps<{
  contact: Contact
}>()

const { t } = useI18n()

const initials = computed(() => {
  const name = props.contact.name ?? ''
  return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2) || '?'
})
</script>

<template>
  <div class="flex items-start gap-4">
    <UAvatar
      :text="initials"
      size="2xl"
      class="shrink-0"
    />
    <div class="min-w-0 flex-1">
      <h1 class="text-xl font-bold truncate">
        {{ contact.name || '—' }}
      </h1>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 mt-1 text-sm text-muted">
        <span v-if="contact.email" class="flex items-center gap-1">
          <UIcon name="i-lucide-mail" class="size-3.5" />
          {{ contact.email }}
        </span>
        <span v-if="contact.phoneNumber" class="flex items-center gap-1">
          <UIcon name="i-lucide-phone" class="size-3.5" />
          {{ contact.phoneNumber }}
        </span>
        <span v-if="contact.identifier" class="flex items-center gap-1">
          <UIcon name="i-lucide-hash" class="size-3.5" />
          {{ contact.identifier }}
        </span>
      </div>
      <p class="text-xs text-dimmed mt-2">
        {{ t('contacts.columns.created') }}: {{ new Date(contact.createdAt).toLocaleDateString() }}
      </p>
    </div>
  </div>
</template>
