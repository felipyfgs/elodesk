<script setup lang="ts">
import type { StepperItem } from '@nuxt/ui'

// Wizard genérico para fluxos de criação de inbox lineares (1-N steps com
// validação Zod + submit POST). Não cobre wizards com efeitos especiais
// (QR-code polling, OAuth, multi-fase) — esses ficam standalone.

const props = defineProps<{
  title: string
  description: string
  steps: StepperItem[]
  cancelTo: string
  validateStep: (step: number) => Promise<boolean>
  submit: () => Promise<void>
  loading: boolean
  submitLabel?: string
}>()

const step = defineModel<number>('step', { default: 0 })

const { t } = useI18n()

const isLastStep = computed(() => step.value >= props.steps.length - 1)

const slotNames = computed(() =>
  props.steps.map(s => s.slot).filter((name): name is string => typeof name === 'string')
)

async function nextStep() {
  if (!(await props.validateStep(step.value))) return
  step.value++
}

async function doSubmit() {
  if (!(await props.validateStep(step.value))) return
  await props.submit()
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h2 class="text-lg font-semibold">
        {{ title }}
      </h2>
      <p class="text-sm text-muted">
        {{ description }}
      </p>
    </div>

    <UStepper v-model="step" :items="steps" :linear="true">
      <template v-for="slotName in slotNames" #[slotName] :key="slotName">
        <slot :name="slotName" />
      </template>
    </UStepper>

    <div class="flex justify-end gap-2">
      <UButton :to="cancelTo" variant="ghost" color="neutral">
        {{ t('common.cancel') }}
      </UButton>
      <UButton v-if="!isLastStep" :disabled="loading" @click="nextStep">
        {{ t('common.next') }}
      </UButton>
      <UButton
        v-else
        type="button"
        :loading="loading"
        @click="doSubmit"
      >
        {{ submitLabel ?? t('common.create') }}
      </UButton>
    </div>
  </div>
</template>
