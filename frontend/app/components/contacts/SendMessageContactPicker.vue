<script setup lang="ts">
import type { Contact } from '~/stores/contacts'

interface ContactItem {
  label: string
  value: string
  // additional fields tolerated
  [key: string]: unknown
}

defineProps<{
  selected: Contact | null
  items: ContactItem[]
  searching: boolean
  canCreateFromTerm: boolean
  createLabel: string
  creating: boolean
}>()

const selectedId = defineModel<string | undefined>('selectedId')

const emit = defineEmits<{
  searchTerm: [value: string]
  createFromTerm: []
  clear: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex items-center gap-3 px-4 py-2.5 min-h-11">
    <span class="text-sm font-medium text-muted shrink-0 w-10">
      {{ t('contactsSendMessage.toLabel') }}:
    </span>
    <UBadge
      v-if="selected"
      color="primary"
      variant="soft"
      size="md"
    >
      <span class="truncate max-w-[14rem]">
        {{ selected.name ?? selected.email ?? selected.phoneNumber ?? '—' }}
      </span>
      <UButton
        icon="i-lucide-x"
        variant="link"
        color="neutral"
        size="xs"
        :padded="false"
        class="ml-1 p-0"
        @click="emit('clear')"
      />
    </UBadge>
    <USelectMenu
      v-else
      v-model="selectedId"
      :items="items"
      value-key="value"
      searchable
      :searchable-placeholder="t('contactsSendMessage.searchContactPlaceholder')"
      :placeholder="t('contactsSendMessage.toPlaceholder')"
      :loading="searching"
      variant="ghost"
      class="flex-1"
      @update:search-term="(v: string) => emit('searchTerm', v)"
    >
      <template #empty="{ searchTerm }">
        <div v-if="canCreateFromTerm" class="p-1">
          <UButton
            :label="createLabel"
            :loading="creating"
            icon="i-lucide-plus"
            color="primary"
            variant="soft"
            block
            @click="emit('createFromTerm')"
          />
        </div>
        <p v-else class="text-sm text-muted text-center py-3 px-2">
          {{ searchTerm ? t('contactsSendMessage.noMatch') : t('contactsSendMessage.typeToSearch') }}
        </p>
      </template>
    </USelectMenu>
  </div>
</template>
