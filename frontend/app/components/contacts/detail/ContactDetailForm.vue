<script setup lang="ts">
import type { Contact } from '~/stores/contacts'
import { parseJsonAttrs } from '~/utils/jsonAttrs'

export interface ContactFormState {
  firstName: string
  lastName: string
  email: string
  phoneNumber: string
  city: string
  country: string
  bio: string
  companyName: string
  socialProfiles: {
    linkedin: string
    facebook: string
    instagram: string
    twitter: string
    github: string
  }
}

const props = defineProps<{
  contact: Contact
}>()

const emit = defineEmits<{
  submit: [state: ContactFormState]
}>()

const { t } = useI18n()

function splitName(name: string | null): { first: string, last: string } {
  const parts = (name ?? '').trim().split(/\s+/)
  if (parts.length === 0 || !parts[0]) return { first: '', last: '' }
  const [first, ...rest] = parts
  return { first, last: rest.join(' ') }
}

function initialState(): ContactFormState {
  const { first, last } = splitName(props.contact.name)
  const attrs = parseJsonAttrs(props.contact.additionalAttributes)
  const social = (attrs.socialProfiles as Record<string, string> | undefined) ?? {}
  return {
    firstName: first,
    lastName: last,
    email: props.contact.email ?? '',
    phoneNumber: props.contact.phoneNumber ?? '',
    city: (attrs.city as string) ?? '',
    country: (attrs.country as string) ?? '',
    bio: (attrs.description as string) ?? '',
    companyName: (attrs.companyName as string) ?? '',
    socialProfiles: {
      linkedin: social.linkedin ?? '',
      facebook: social.facebook ?? '',
      instagram: social.instagram ?? '',
      twitter: social.twitter ?? '',
      github: social.github ?? ''
    }
  }
}

const state = reactive<ContactFormState>(initialState())
const emailError = ref('')

watch(() => props.contact.id, () => {
  Object.assign(state, initialState())
})

const fullName = computed(() => `${state.firstName} ${state.lastName}`.trim())
const isInvalid = computed(() => !state.firstName.trim() || !!emailError.value)

function validateEmail() {
  if (!state.email) {
    emailError.value = ''
    return
  }
  const ok = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(state.email)
  emailError.value = ok ? '' : t('contacts.card.form.emailInvalid')
}

defineExpose({
  submit: () => {
    validateEmail()
    if (isInvalid.value) return
    emit('submit', JSON.parse(JSON.stringify(state)) as ContactFormState)
  },
  fullName,
  isInvalid
})

const socialIcons: Record<keyof ContactFormState['socialProfiles'], string> = {
  linkedin: 'i-simple-icons-linkedin',
  facebook: 'i-simple-icons-facebook',
  instagram: 'i-simple-icons-instagram',
  twitter: 'i-simple-icons-x',
  github: 'i-simple-icons-github'
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <section>
      <h3 class="text-sm font-medium text-highlighted mb-3">
        {{ t('contactDetail.sections.details') }}
      </h3>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <UFormField name="firstName">
          <UInput
            v-model="state.firstName"
            :placeholder="t('contacts.form.firstNamePlaceholder')"
            class="w-full"
          />
        </UFormField>
        <UFormField name="lastName">
          <UInput
            v-model="state.lastName"
            :placeholder="t('contacts.form.lastNamePlaceholder')"
            class="w-full"
          />
        </UFormField>
        <UFormField name="email" :error="emailError">
          <UInput
            v-model="state.email"
            type="email"
            :placeholder="t('contacts.form.emailPlaceholder')"
            class="w-full"
            @blur="validateEmail"
          />
        </UFormField>
        <UFormField name="phoneNumber">
          <PhoneNumberInput v-model="state.phoneNumber" />
        </UFormField>
        <UFormField name="city">
          <UInput
            v-model="state.city"
            :placeholder="t('contacts.form.cityPlaceholder')"
            class="w-full"
          />
        </UFormField>
        <UFormField name="country">
          <UInput
            v-model="state.country"
            :placeholder="t('contacts.form.countryPlaceholder')"
            class="w-full"
          />
        </UFormField>
        <UFormField name="bio" class="sm:col-span-2">
          <UTextarea
            v-model="state.bio"
            :placeholder="t('contacts.form.bioPlaceholder')"
            :rows="2"
            class="w-full"
          />
        </UFormField>
        <UFormField name="companyName" class="sm:col-span-2">
          <UInput
            v-model="state.companyName"
            :placeholder="t('contacts.form.companyPlaceholder')"
            class="w-full"
          />
        </UFormField>
      </div>
    </section>

    <section>
      <h3 class="text-sm font-medium text-highlighted mb-3">
        {{ t('contacts.form.socialProfiles') }}
      </h3>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="(_, key) in state.socialProfiles"
          :key="key"
          class="flex items-center gap-2 px-3 h-9 rounded-md bg-elevated"
        >
          <UIcon :name="socialIcons[key]" class="size-4 text-muted shrink-0" />
          <input
            v-model="state.socialProfiles[key]"
            :placeholder="t(`contacts.form.${key}Placeholder`)"
            class="bg-transparent outline-none text-sm placeholder:text-dimmed min-w-[140px]"
          >
        </div>
      </div>
    </section>
  </div>
</template>
