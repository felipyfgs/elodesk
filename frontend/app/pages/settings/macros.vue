<script setup lang="ts">
import MacroBuilder from '~/components/settings/macros/MacroBuilder.vue'
import { ConfirmModal } from '#components'
import type { Macro } from '~/stores/macros'
import { useMacrosStore } from '~/stores/macros'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const store = useMacrosStore()
const confirm = useOverlay().create(ConfirmModal)

const builderOpen = ref(false)
const editing = ref<Macro | null>(null)

onMounted(() => {
  store.fetch()
})

function onNew() {
  editing.value = null
  builderOpen.value = true
}

function onEdit(m: Macro) {
  editing.value = m
  builderOpen.value = true
}

async function onDelete(m: Macro) {
  const confirmed = await confirm.open({
    title: t('common.delete'),
    confirmLabel: t('common.delete'),
    itemName: m.name
  })
  if (!confirmed) return
  await store.remove(m.id)
}
</script>

<template>
  <UPageCard :title="t('settings.macros.title')" variant="subtle">
    <template #footer>
      <div class="flex justify-end">
        <UButton icon="i-lucide-plus" @click="onNew">
          {{ t('settings.macros.new') }}
        </UButton>
      </div>
    </template>

    <div class="border border-default rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted text-left">
          <tr>
            <th class="px-4 py-2 font-medium">
              {{ t('settings.general.name') }}
            </th>
            <th class="px-4 py-2 font-medium">
              {{ t('settings.macros.actions') }}
            </th>
            <th class="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-if="store.loading">
            <td colspan="3" class="px-4 py-6 text-center text-muted">
              …
            </td>
          </tr>
          <tr v-else-if="store.items.length === 0">
            <td colspan="3" class="px-4 py-6 text-center text-muted">
              {{ t('settings.macros.empty') }}
            </td>
          </tr>
          <tr v-for="m in store.items" :key="m.id" class="border-t border-default">
            <td class="px-4 py-2">
              {{ m.name }}
            </td>
            <td class="px-4 py-2 text-muted">
              <UBadge :label="m.visibility" variant="soft" size="sm" />
            </td>
            <td class="px-4 py-2 text-right">
              <UButtonGroup size="xs">
                <UButton variant="ghost" icon="i-lucide-pencil" @click="onEdit(m)" />
                <UButton
                  variant="ghost"
                  color="error"
                  icon="i-lucide-trash"
                  @click="onDelete(m)"
                />
              </UButtonGroup>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <MacroBuilder v-model:open="builderOpen" :macro="editing" />
  </UPageCard>
</template>
