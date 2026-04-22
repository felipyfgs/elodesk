<script setup lang="ts">
import parsePhoneNumber, {
  AsYouType,
  getExampleNumber,
  isValidPhoneNumber,
  validatePhoneNumberLength,
  type CountryCode
} from 'libphonenumber-js'
import examples from 'libphonenumber-js/mobile/examples'
import { countries, type Country } from '~/utils/countries'

const props = defineProps<{
  placeholder?: string
  disabled?: boolean
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  defaultCountry?: string
  required?: boolean
}>()

const modelValue = defineModel<string>({ default: '' })
const emit = defineEmits<{
  'update:valid': [value: boolean]
}>()

const { t } = useI18n()

const DEFAULT_COUNTRY = (props.defaultCountry ?? 'BR') as CountryCode

const countryCode = ref<CountryCode>(DEFAULT_COUNTRY)
const phone = ref<string>('')

const activeCountry = computed<Country | undefined>(() =>
  countries.find(c => c.id === countryCode.value)
)

const dialCode = computed(() => activeCountry.value?.dial_code ?? '+55')

const dynamicPlaceholder = computed(() => {
  if (props.placeholder) return props.placeholder
  const example = getExampleNumber(countryCode.value, examples)
  return example?.formatNational() ?? t('phoneInput.placeholder')
})

const countryItems = computed(() =>
  countries.map(c => ({
    label: c.name,
    value: c.id,
    emoji: c.emoji,
    dialCode: c.dial_code
  }))
)

function detect(raw: string) {
  if (!raw) return null
  const intl = raw.startsWith('+') ? raw : `+${raw.replace(/\D/g, '')}`
  const asIntl = parsePhoneNumber(intl)
  if (asIntl?.country && asIntl.isValid()) return asIntl
  return parsePhoneNumber(raw, countryCode.value) ?? null
}

function formatNational(digits: string, country: CountryCode): string {
  if (!digits) return ''
  return new AsYouType(country).input(digits)
}

function getCurrentE164(): string {
  if (!phone.value) return ''
  const formatter = new AsYouType(countryCode.value)
  formatter.input(phone.value)
  return formatter.getNumberValue() ?? ''
}

function syncModel() {
  if (!phone.value) {
    modelValue.value = ''
    return
  }
  const e164 = getCurrentE164()
  modelValue.value = e164 || `${dialCode.value}${phone.value.replace(/\D/g, '')}`
}

function hydrate(raw: string) {
  if (!raw) {
    phone.value = ''
    return
  }
  const parsed = detect(raw)
  if (parsed?.country) {
    countryCode.value = parsed.country
    phone.value = parsed.formatNational()
    return
  }
  phone.value = formatNational(raw.replace(/\D/g, ''), countryCode.value)
}

function onPhoneInput(raw: string) {
  phone.value = formatNational(raw.replace(/\D/g, ''), countryCode.value)
  syncModel()
}

function onCountryChange(code: string) {
  const next = code as CountryCode
  countryCode.value = next
  if (phone.value) {
    phone.value = formatNational(phone.value.replace(/\D/g, ''), next)
    syncModel()
  }
}

const touched = ref(false)

const validationError = computed<string | null>(() => {
  if (!touched.value) return null
  if (!phone.value) {
    return props.required ? t('phoneInput.required') : null
  }
  const e164 = getCurrentE164()
  if (!e164) return t('phoneInput.invalid')
  const lengthError = validatePhoneNumberLength(e164)
  if (lengthError === 'TOO_SHORT') return t('phoneInput.tooShort')
  if (lengthError === 'TOO_LONG') return t('phoneInput.tooLong')
  if (lengthError === 'INVALID_LENGTH' || lengthError === 'NOT_A_NUMBER') {
    return t('phoneInput.invalid')
  }
  if (!isValidPhoneNumber(e164)) return t('phoneInput.invalid')
  return null
})

const isValid = computed(() => {
  if (!phone.value) return !props.required
  const e164 = getCurrentE164()
  return !!e164 && isValidPhoneNumber(e164)
})

function onBlur() {
  touched.value = true
}

watch(isValid, v => emit('update:valid', v), { immediate: true })

defineExpose({
  isValid,
  validate: () => {
    touched.value = true
    return isValid.value
  }
})

hydrate(modelValue.value)

watch(modelValue, (newValue, oldValue) => {
  if (newValue === oldValue) return
  if (newValue === getCurrentE164()) return
  hydrate(newValue)
})
</script>

<template>
  <div>
    <UFieldGroup :size="size">
      <USelectMenu
        :model-value="countryCode"
        :items="countryItems"
        value-key="value"
        :search-input="{
          placeholder: t('phoneInput.searchPlaceholder'),
          icon: 'i-lucide-search'
        }"
        :filter-fields="['label', 'value', 'dialCode']"
        :content="{ align: 'start' }"
        :disabled="disabled"
        :color="validationError ? 'error' : undefined"
        :highlight="!!validationError"
        :ui="{
          base: 'pe-8',
          content: 'w-64',
          placeholder: 'hidden',
          trailingIcon: 'size-4'
        }"
        trailing-icon="i-lucide-chevrons-up-down"
        @update:model-value="onCountryChange"
      >
        <span class="size-5 flex items-center text-lg">
          {{ activeCountry?.emoji || '🌐' }}
        </span>
        <template #item-leading="{ item }">
          <span class="size-5 flex items-center text-lg">{{ item.emoji }}</span>
        </template>
        <template #item-label="{ item }">
          {{ item.label }} ({{ item.dialCode }})
        </template>
      </USelectMenu>

      <UInput
        :model-value="phone"
        type="tel"
        :placeholder="dynamicPlaceholder"
        :disabled="disabled"
        :color="validationError ? 'error' : undefined"
        :highlight="!!validationError"
        :style="{ '--dial-code-length': `${dialCode.length + 2}ch` }"
        :ui="{
          base: 'ps-(--dial-code-length) w-full',
          leading: 'pointer-events-none text-muted'
        }"
        @update:model-value="onPhoneInput"
        @blur="onBlur"
      >
        <template #leading>
          <span class="pe-1.5">{{ dialCode }}</span>
        </template>
      </UInput>
    </UFieldGroup>
    <p v-if="validationError" class="text-xs text-error mt-1">
      {{ validationError }}
    </p>
  </div>
</template>
