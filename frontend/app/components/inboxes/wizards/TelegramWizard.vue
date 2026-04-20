<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { telegramStepSetupSchema, telegramStepCredentialsSchema, type TelegramInboxForm } from '~/schemas/inboxes/telegram'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<TelegramInboxForm>({
  name: '',
  botToken: ''
})

const loading = ref(false)
const step = ref(0)

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-send', slot: 'setup' },
  { title: t('inboxes.wizards.credentials'), icon: 'i-lucide-key', slot: 'credentials' }
]

const setupFormRef = ref()
const credentialsFormRef = ref()

async function validateCurrentStep(): Promise<boolean> {
  if (step.value === 0) {
    const { error } = await telegramStepSetupSchema.safeParseAsync({
      name: state.name
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
    const { error } = await telegramStepCredentialsSchema.safeParseAsync({
      botToken: state.botToken
    })
    if (error) {
      credentialsFormRef.value?.setErrors(
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
    const res = await api<{ id: number }>(`/accounts/${auth.account.id}/inboxes/telegram`, {
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
          {{ t('inboxes.channels.telegram') }}
        </h2>
        <p class="text-sm text-muted">
          {{ t('inboxes.wizards.telegram.description') }}
        </p>
      </div>
    </div>

    <UStepper v-model="step" :items="stepperItems" :linear="true">
      <template #setup="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="setupFormRef"
            :schema="telegramStepSetupSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.name')" name="name" required>
              <UInput v-model="state.name" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>

      <template #credentials="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="credentialsFormRef"
            :schema="telegramStepCredentialsSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.telegram.botToken')" name="botToken" required>
              <UTextarea v-model="state.botToken" :rows="2" />
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
