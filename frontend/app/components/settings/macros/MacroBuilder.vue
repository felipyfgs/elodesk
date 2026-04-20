<script setup lang="ts">
import MacroActionsList from './MacroActionsList.vue'
import MacroConditionsEditor from './MacroConditionsEditor.vue'
import { macroSchema, type MacroForm } from '~/schemas/settings/macros'
import { useMacrosStore, type Macro } from '~/stores/macros'

const props = defineProps<{ open: boolean, macro?: Macro | null }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const { t } = useI18n()
const toast = useToast()
const store = useMacrosStore()

const state = reactive<Partial<MacroForm>>({
  name: '',
  visibility: 'account',
  conditions: {},
  actions: []
})
const loading = ref(false)

watch(() => props.open, (open) => {
  if (!open) return
  if (props.macro) {
    state.name = props.macro.name
    state.visibility = props.macro.visibility as 'account' | 'personal'
    try {
      state.conditions = typeof props.macro.conditions === 'string' ? JSON.parse(props.macro.conditions) : (props.macro.conditions as Record<string, unknown>) ?? {}
    } catch { state.conditions = {} }
    try {
      state.actions = typeof props.macro.actions === 'string' ? JSON.parse(props.macro.actions) : (props.macro.actions as MacroForm['actions']) ?? []
    } catch { state.actions = [] }
  } else {
    state.name = ''
    state.visibility = 'account'
    state.conditions = {}
    state.actions = []
  }
})

async function onSubmit() {
  loading.value = true
  try {
    await store.save({
      id: props.macro?.id,
      name: state.name,
      visibility: state.visibility,
      conditions: state.conditions,
      actions: state.actions
    } as Partial<Macro>)
    toast.add({ title: t('common.save'), color: 'success' })
    emit('update:open', false)
  } catch {
    toast.add({ title: t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal :open="props.open" :title="props.macro ? t('common.edit') : t('settings.macros.new')" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-6">
        <UForm
          :schema="macroSchema"
          :state="state"
          class="space-y-4"
          @submit="onSubmit"
        >
          <UFormField :label="t('settings.general.name')" name="name">
            <UInput v-model="state.name" />
          </UFormField>
          <UFormField :label="t('settings.macros.conditions')" name="conditions">
            <MacroConditionsEditor v-model="state.conditions as Record<string, unknown>" />
          </UFormField>
          <UFormField :label="t('settings.macros.actions')" name="actions">
            <MacroActionsList v-model="state.actions as MacroForm['actions']" />
          </UFormField>
          <div class="flex justify-end gap-2">
            <UButton variant="outline" @click="emit('update:open', false)">
              {{ t('common.cancel') }}
            </UButton>
            <UButton type="submit" :loading="loading">
              {{ t('common.save') }}
            </UButton>
          </div>
        </UForm>
      </div>
    </template>
  </UModal>
</template>
