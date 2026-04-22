<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'
import { apiInboxSchema, type ApiInboxForm } from '~/schemas/inboxes/api'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const state = reactive<ApiInboxForm>({
  name: ''
})

const loading = ref(false)
const step = ref(0)

const stepperItems: StepperItem[] = [
  { title: t('inboxes.wizards.setup'), icon: 'i-lucide-code', slot: 'setup' }
]

const setupFormRef = ref()

async function validateCurrentStep(): Promise<boolean> {
  const { error } = await apiInboxSchema.safeParseAsync({
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

async function submit() {
  if (!auth.account?.id) return
  if (!(await validateCurrentStep())) return
  loading.value = true
  try {
    const res = await api<{
      id: number
      identifier: string
      apiToken: string
      hmacToken: string
    }>(`/accounts/${auth.account.id}/inboxes`, {
      method: 'POST',
      body: state
    })
    toast.add({ title: t('common.success'), color: 'success' })
    router.push(`/accounts/${auth.account?.id}/inboxes/${res.id}`)
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
    <div>
      <h2 class="text-lg font-semibold">
        {{ t('inboxes.channels.api') }}
      </h2>
      <p class="text-sm text-muted">
        {{ t('inboxes.wizards.api.description') }}
      </p>
    </div>

    <UStepper v-model="step" :items="stepperItems" :linear="true">
      <template #setup="{}">
        <UPageCard variant="subtle">
          <UForm
            ref="setupFormRef"
            :schema="apiInboxSchema"
            :state="state"
            class="flex flex-col gap-4"
          >
            <UFormField :label="t('inboxes.wizards.name')" name="name" required>
              <UInput v-model="state.name" />
            </UFormField>
          </UForm>
        </UPageCard>
      </template>
    </UStepper>

    <div class="flex justify-end gap-2">
      <UButton :to="`/accounts/${auth.account?.id}/inboxes/new`" variant="ghost" color="neutral">
        {{ t('common.cancel') }}
      </UButton>
      <UButton v-if="!isLastStep" :disabled="loading" @click="step++">
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
