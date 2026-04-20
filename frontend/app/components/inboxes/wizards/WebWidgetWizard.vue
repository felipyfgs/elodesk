<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { webWidgetStepSetupSchema, webWidgetStepAppearanceSchema, type WebWidgetInboxForm } from '~/schemas/inboxes/webWidget'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<WebWidgetInboxForm>({
  name: '',
  websiteUrl: '',
  widgetColor: '',
  welcomeTitle: '',
  welcomeTagline: '',
  replyTime: 'in_a_few_minutes'
})

const loading = ref(false)
const step = ref(0)

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-globe', slot: 'setup' },
  { title: t('inboxes.wizards.appearance'), icon: 'i-lucide-palette', slot: 'appearance' }
]

const setupFormRef = ref()
const appearanceFormRef = ref()

async function validateCurrentStep(): Promise<boolean> {
  if (step.value === 0) {
    const { error } = await webWidgetStepSetupSchema.safeParseAsync({
      name: state.name,
      websiteUrl: state.websiteUrl
    })
    if (error) {
      setupFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }

  if (step.value === 1) {
    const { error } = await webWidgetStepAppearanceSchema.safeParseAsync({
      widgetColor: state.widgetColor,
      welcomeTitle: state.welcomeTitle,
      welcomeTagline: state.welcomeTagline,
      replyTime: state.replyTime
    })
    if (error) {
      appearanceFormRef.value?.setErrors(
        error.issues.map(i => ({ message: i.message, path: i.path.join('.') }))
      )
      return false
    }
    return true
  }

  return true
}

async function nextStep() {
  if (!(await validateCurrentStep())) return
  step.value++
}

async function submit() {
  if (!auth.account?.id) return
  if (!(await validateCurrentStep())) return
  loading.value = true
  try {
    const res = await api<{ id: number }>(`/accounts/${auth.account.id}/inboxes/web_widget`, {
      method: 'POST',
      body: state
    })
    toast.add({ title: t('common.success'), color: 'success' })
    router.push(`/inboxes/${res.id}`)
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    loading.value = false
  }
}

const isLastStep = computed(() => step.value >= stepperItems.length - 1)
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex items-center gap-3">
      <UButton
        icon="i-lucide-arrow-left"
        variant="ghost"
        color="neutral"
        to="/inboxes/new"
      />
      <div>
        <h2 class="text-lg font-semibold">
          {{ t('inboxes.channels.web_widget') }}
        </h2>
        <p class="text-sm text-muted">
          {{ t('inboxes.wizards.webWidget.description') }}
        </p>
      </div>
    </div>

    <UStepper v-model="step" :items="stepperItems" :linear="true">
      <template #setup="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="setupFormRef"
            :schema="webWidgetStepSetupSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.name')" name="name" required>
              <UInput v-model="state.name" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.webWidget.websiteUrl')" name="websiteUrl" required>
              <UInput v-model="state.websiteUrl" placeholder="https://" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #appearance="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="appearanceFormRef"
            :schema="webWidgetStepAppearanceSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.webWidget.widgetColor')" name="widgetColor">
              <UInput v-model="state.widgetColor" placeholder="#0084FF" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.webWidget.welcomeTitle')" name="welcomeTitle">
              <UInput v-model="state.welcomeTitle" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.webWidget.welcomeTagline')" name="welcomeTagline">
              <UTextarea v-model="state.welcomeTagline" :rows="2" />
            </UFormField>

            <UFormField :label="t('inboxes.wizards.webWidget.replyTime')" name="replyTime">
              <USelect
                v-model="state.replyTime"
                :options="[
                  { value: 'in_a_few_minutes', label: t('inboxes.wizards.webWidget.replyTimes.in_a_few_minutes') },
                  { value: 'in_a_few_hours', label: t('inboxes.wizards.webWidget.replyTimes.in_a_few_hours') },
                  { value: 'in_a_day', label: t('inboxes.wizards.webWidget.replyTimes.in_a_day') }
                ]"
                value-key="value"
                option-attribute="label"
              />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>
    </UStepper>

    <div class="flex justify-end gap-2">
      <UButton to="/inboxes/new" variant="ghost" color="neutral">
        {{ t('common.cancel') }}
      </UButton>
      <UButton v-if="!isLastStep" :disabled="loading" @click="nextStep">
        {{ t('common.next') }}
      </UButton>
      <UButton
        v-else
        type="button"
        :loading="loading"
        @click="submit"
      >
        {{ t('common.create') }}
      </UButton>
    </div>
  </div>
</template>
