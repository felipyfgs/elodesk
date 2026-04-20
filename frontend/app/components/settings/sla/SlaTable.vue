<script setup lang="ts">
import type { SlaPolicy } from '~/stores/sla'

defineProps<{ items: SlaPolicy[], loading: boolean }>()
const emit = defineEmits<{ edit: [policy: SlaPolicy], remove: [policy: SlaPolicy] }>()

const { t } = useI18n()
</script>

<template>
  <div class="border border-default rounded-lg overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-muted text-left">
        <tr>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.general.name') }}
          </th>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.sla.firstResponse') }}
          </th>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.sla.resolution') }}
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
            {{ t('settings.sla.empty') }}
          </td>
        </tr>
        <tr v-for="p in items" :key="p.id" class="border-t border-default">
          <td class="px-4 py-2">
            {{ p.name }}
          </td>
          <td class="px-4 py-2 text-muted">
            {{ p.firstResponseMinutes }} min
          </td>
          <td class="px-4 py-2 text-muted">
            {{ p.resolutionMinutes }} min
          </td>
          <td class="px-4 py-2 text-right">
            <UButtonGroup size="xs">
              <UButton variant="ghost" icon="i-lucide-pencil" @click="emit('edit', p)" />
              <UButton
                variant="ghost"
                color="error"
                icon="i-lucide-trash"
                @click="emit('remove', p)"
              />
            </UButtonGroup>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
