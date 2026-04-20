<script setup lang="ts">
interface Row {
  id: number
  displayId: number
  inboxId: number
  assigneeId?: number | null
  status: number
  createdAt: string
}

defineProps<{ items: Row[], loading: boolean }>()
const { t: _t } = useI18n()

function statusLabel(s: number) {
  return ['open', 'resolved', 'pending', 'snoozed'][s] ?? String(s)
}
</script>

<template>
  <div class="border border-default rounded-lg overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-muted text-left">
        <tr>
          <th class="px-4 py-2 font-medium">
            #
          </th>
          <th class="px-4 py-2 font-medium">
            Inbox
          </th>
          <th class="px-4 py-2 font-medium">
            Assignee
          </th>
          <th class="px-4 py-2 font-medium">
            Status
          </th>
          <th class="px-4 py-2 font-medium">
            Created
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td colspan="5" class="px-4 py-6 text-center text-muted">
            …
          </td>
        </tr>
        <tr v-else-if="items.length === 0">
          <td colspan="5" class="px-4 py-6 text-center text-muted">
            —
          </td>
        </tr>
        <tr v-for="r in items" :key="r.id" class="border-t border-default">
          <td class="px-4 py-2">
            {{ r.displayId }}
          </td>
          <td class="px-4 py-2 text-muted">
            {{ r.inboxId }}
          </td>
          <td class="px-4 py-2 text-muted">
            {{ r.assigneeId ?? '-' }}
          </td>
          <td class="px-4 py-2">
            <UBadge :label="statusLabel(r.status)" variant="soft" size="sm" />
          </td>
          <td class="px-4 py-2 text-muted text-xs">
            {{ new Date(r.createdAt).toLocaleString() }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
