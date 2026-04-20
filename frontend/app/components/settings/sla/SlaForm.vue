<script setup lang="ts">
import SlaBindingsPicker from './SlaBindingsPicker.vue'
import { slaSchema, type SlaForm as SlaFormType } from '~/schemas/settings/sla'
import { useSlaStore, type SlaPolicy } from '~/stores/sla'

const props = defineProps<{ open: boolean, policy?: SlaPolicy | null }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const { t } = useI18n()
const toast = useToast()
const store = useSlaStore()

const state = reactive<Partial<SlaFormType>>({
  name: '', firstResponseMinutes: 60, resolutionMinutes: 1440,
  businessHoursOnly: false, inboxIds: [], labelIds: []
})
const loading = ref(false)

watch(() => props.open, (o) => {
  if (!o) return
  if (props.policy) {
    state.name = props.policy.name
    state.firstResponseMinutes = props.policy.firstResponseMinutes
    state.resolutionMinutes = props.policy.resolutionMinutes
    state.businessHoursOnly = props.policy.businessHoursOnly
    state.inboxIds = []
    state.labelIds = []
  } else {
    state.name = ''
    state.firstResponseMinutes = 60
    state.resolutionMinutes = 1440
    state.businessHoursOnly = false
    state.inboxIds = []
    state.labelIds = []
  }
})

async function onSubmit() {
  loading.value = true
  try {
    await store.save({
      id: props.policy?.id,
      name: state.name,
      firstResponseMinutes: state.firstResponseMinutes,
      resolutionMinutes: state.resolutionMinutes,
      businessHoursOnly: state.businessHoursOnly
    } as Partial<SlaPolicy>)
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
  <USlideover :open="props.open" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-6 space-y-4">
        <h3 class="text-lg font-semibold">
          {{ props.policy ? t('common.edit') : t('settings.sla.new') }}
        </h3>
        <UForm
          :schema="slaSchema"
          :state="state"
          class="space-y-4"
          @submit="onSubmit"
        >
          <UFormField :label="t('settings.general.name')" name="name">
            <UInput v-model="state.name" />
          </UFormField>
          <UFormField :label="t('settings.sla.firstResponse') + ' (min)'" name="firstResponseMinutes">
            <UInput v-model.number="state.firstResponseMinutes" type="number" min="1" />
          </UFormField>
          <UFormField :label="t('settings.sla.resolution') + ' (min)'" name="resolutionMinutes">
            <UInput v-model.number="state.resolutionMinutes" type="number" min="1" />
          </UFormField>
          <UFormField name="businessHoursOnly">
            <UCheckbox v-model="state.businessHoursOnly" :label="t('settings.sla.binding')" />
          </UFormField>
          <SlaBindingsPicker
            :inbox-ids="state.inboxIds ?? []"
            :label-ids="state.labelIds ?? []"
            @update:inbox-ids="state.inboxIds = $event"
            @update:label-ids="state.labelIds = $event"
          />
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
  </USlideover>
</template>
