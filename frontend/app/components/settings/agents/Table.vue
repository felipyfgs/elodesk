<script setup lang="ts">
import type { Agent } from '~/stores/agents'

defineProps<{ items: Agent[], loading: boolean }>()
const emit = defineEmits<{ edit: [agent: Agent], remove: [agent: Agent] }>()

const { t } = useI18n()

function roleLabel(role: number) {
  if (role === 2) return 'Owner'
  if (role === 1) return 'Admin'
  return 'Agent'
}
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
            {{ t('settings.general.email') }}
          </th>
          <th class="px-4 py-2 font-medium">
            {{ t('settings.agents.role') }}
          </th>
          <th class="px-4 py-2" />
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td colspan="4" class="px-4 py-6 text-center text-muted">
            {{ t('common.loading') ?? 'Loading...' }}
          </td>
        </tr>
        <tr v-else-if="items.length === 0">
          <td colspan="4" class="px-4 py-6 text-center text-muted">
            {{ t('settings.agents.empty') }}
          </td>
        </tr>
        <tr v-for="agent in items" :key="agent.id" class="border-t border-default">
          <td class="px-4 py-2 flex items-center gap-2">
            <UAvatar :alt="agent.name" size="sm" />
            <span>{{ agent.name }}</span>
          </td>
          <td class="px-4 py-2 text-muted">
            {{ agent.email }}
          </td>
          <td class="px-4 py-2">
            <UBadge :label="roleLabel(agent.role)" variant="soft" />
          </td>
          <td class="px-4 py-2 text-right">
            <UFieldGroup size="xs">
              <UButton variant="ghost" icon="i-lucide-pencil" @click="emit('edit', agent)" />
              <UButton
                variant="ghost"
                color="error"
                icon="i-lucide-trash"
                @click="emit('remove', agent)"
              />
            </UFieldGroup>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
