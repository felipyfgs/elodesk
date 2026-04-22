<script setup lang="ts">
import type { OutboundWebhook } from '~/stores/webhooks'

defineProps<{ items: OutboundWebhook[], loading: boolean }>()
const emit = defineEmits<{ edit: [hook: OutboundWebhook], remove: [hook: OutboundWebhook] }>()

const { t } = useI18n()
</script>

<template>
  <div class="border border-default rounded-lg overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-muted text-left">
        <tr>
          <th class="px-4 py-2 font-medium">
            URL
          </th>
          <th class="px-4 py-2 font-medium">
            Subscriptions
          </th>
          <th class="px-4 py-2 font-medium">
            Status
          </th>
          <th class="px-4 py-2" />
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td colspan="4" class="px-4 py-6 text-center text-muted">
            …
          </td>
        </tr>
        <tr v-else-if="items.length === 0">
          <td colspan="4" class="px-4 py-6 text-center text-muted">
            {{ t('settings.integrations.empty') }}
          </td>
        </tr>
        <tr v-for="h in items" :key="h.id" class="border-t border-default">
          <td class="px-4 py-2 font-mono text-xs">
            {{ h.url }}
          </td>
          <td class="px-4 py-2 text-muted">
            {{ (h.subscriptions ?? []).join(', ') }}
          </td>
          <td class="px-4 py-2">
            <UBadge
              :label="h.isActive ? 'active' : 'inactive'"
              :color="h.isActive ? 'success' : 'neutral'"
              variant="soft"
              size="sm"
            />
          </td>
          <td class="px-4 py-2 text-right">
            <UFieldGroup size="xs">
              <UButton variant="ghost" icon="i-lucide-pencil" @click="emit('edit', h)" />
              <UButton
                variant="ghost"
                color="error"
                icon="i-lucide-trash"
                @click="emit('remove', h)"
              />
            </UFieldGroup>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
