<script setup lang="ts">
import SlaTable from '~/components/settings/sla/SlaTable.vue'
import SlaForm from '~/components/settings/sla/SlaForm.vue'
import { ConfirmModal } from '#components'
import type { SlaPolicy } from '~/stores/sla'
import { useSlaStore } from '~/stores/sla'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const store = useSlaStore()
const confirm = useOverlay().create(ConfirmModal)

const formOpen = ref(false)
const editing = ref<SlaPolicy | null>(null)

onMounted(() => {
  store.fetch()
})

function onNew() {
  editing.value = null
  formOpen.value = true
}

function onEdit(p: SlaPolicy) {
  editing.value = p
  formOpen.value = true
}

function onRemove(p: SlaPolicy) {
  confirm.open({
    title: t('common.delete'),
    confirmLabel: t('common.delete'),
    itemName: p.name
  }).then(async (ok) => {
    if (!ok) return
    await store.remove(p.id)
  })
}
</script>

<template>
  <UPageCard :title="t('settings.sla.title')" variant="subtle">
    <template #footer>
      <div class="flex justify-end">
        <UButton icon="i-lucide-plus" @click="onNew">
          {{ t('settings.sla.new') }}
        </UButton>
      </div>
    </template>

    <SlaTable
      :items="store.items"
      :loading="store.loading"
      @edit="onEdit"
      @remove="onRemove"
    />
    <SlaForm v-model:open="formOpen" :policy="editing" />
  </UPageCard>
</template>
