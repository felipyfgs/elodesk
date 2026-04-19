<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useLabelsStore, type Label } from '~/stores/labels'

const props = defineProps<{
  contactId: string
  conversationId?: string
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useLabelsStore()

const loading = ref(false)
const appliedLabels = ref<Label[]>([])

const availableLabels = computed(() =>
  store.list.filter(l => !appliedLabels.value.some(a => a.id === l.id))
)

async function loadLabels() {
  if (!auth.account?.id) return
  const res = await api<Label[]>(`/accounts/${auth.account.id}/contacts/${props.contactId}/labels`)
  appliedLabels.value = res
}

async function applyLabel(label: Label) {
  if (!auth.account?.id) return
  await api(`/accounts/${auth.account.id}/contacts/${props.contactId}/labels`, {
    method: 'POST',
    body: { label_id: label.id }
  })
  appliedLabels.value.push(label)
}

async function removeLabel(label: Label) {
  if (!auth.account?.id) return
  await api(`/accounts/${auth.account.id}/contacts/${props.contactId}/labels/${label.id}`, {
    method: 'DELETE'
  })
  appliedLabels.value = appliedLabels.value.filter(l => l.id !== label.id)
}

async function ensureStoreLoaded() {
  if (!store.list.length) {
    const list = await api<Label[]>('/labels')
    store.setAll(list)
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadLabels(), ensureStoreLoaded()])
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <div v-if="loading" class="text-sm text-muted">
      {{ t('common.loading') }}
    </div>

    <div v-else class="space-y-3">
      <div class="flex flex-wrap gap-2">
        <UBadge
          v-for="label in appliedLabels"
          :key="label.id"
          :style="{ backgroundColor: label.color, color: '#fff' }"
          class="cursor-pointer"
          @click="removeLabel(label)"
        >
          {{ label.title }}
          <UIcon name="i-lucide-x" class="ms-1 size-3" />
        </UBadge>
        <span v-if="!appliedLabels.length" class="text-sm text-muted">
          {{ t('common.noResults') }}
        </span>
      </div>

      <UDropdownMenu
        v-if="availableLabels.length"
        :items="[availableLabels.map(l => ({
          label: l.title,
          icon: 'i-lucide-tag',
          onSelect: () => applyLabel(l)
        }))]"
      >
        <UButton variant="outline" size="xs" icon="i-lucide-plus">
          {{ t('labels.create') }}
        </UButton>
      </UDropdownMenu>
    </div>
  </div>
</template>
