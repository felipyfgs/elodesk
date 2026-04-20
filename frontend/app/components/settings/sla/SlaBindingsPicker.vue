<script setup lang="ts">
const props = defineProps<{ inboxIds: number[], labelIds: number[] }>()
const emit = defineEmits<{ 'update:inboxIds': [ids: number[]], 'update:labelIds': [ids: number[]] }>()

const { t: _t } = useI18n()

const inboxText = computed({
  get: () => props.inboxIds.join(', '),
  set: (v: string) => emit('update:inboxIds', parseIds(v))
})
const labelText = computed({
  get: () => props.labelIds.join(', '),
  set: (v: string) => emit('update:labelIds', parseIds(v))
})

function parseIds(v: string): number[] {
  return v.split(',').map(s => Number(s.trim())).filter(n => !Number.isNaN(n) && n > 0)
}
</script>

<template>
  <div class="space-y-2">
    <UFormField label="Inbox IDs">
      <UInput v-model="inboxText" placeholder="1, 2, 3" />
    </UFormField>
    <UFormField label="Label IDs">
      <UInput v-model="labelText" placeholder="1, 2, 3" />
    </UFormField>
  </div>
</template>
