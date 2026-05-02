<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'
import type { Contact } from '~/stores/contacts'
import { countries } from '~/utils/countries'
import { parseJsonAttrs } from '~/utils/jsonAttrs'

const props = defineProps<{
  contact: Contact
  loading?: boolean
}>()

const emit = defineEmits<{
  update: [payload: {
    id: string
    name?: string
    email?: string
    phone_number?: string
    additional_attributes?: Record<string, unknown>
  }]
}>()

const { t } = useI18n()

type Schema = {
  firstName: string
  lastName?: string
  email?: string
  phoneNumber?: string
  city?: string
  country?: string
  bio?: string
  companyName?: string
  linkedin?: string
  facebook?: string
  instagram?: string
  twitter?: string
  github?: string
}

const schema = computed(() => z.object({
  firstName: z.string().min(1, t('contacts.card.form.nameRequired')).max(120),
  lastName: z.string().max(120).optional().or(z.literal('')),
  email: z.string().email(t('contacts.card.form.emailInvalid')).optional().or(z.literal('')),
  phoneNumber: z.string().max(30).optional().or(z.literal('')),
  city: z.string().max(120).optional().or(z.literal('')),
  country: z.string().max(120).optional().or(z.literal('')),
  bio: z.string().max(500).optional().or(z.literal('')),
  companyName: z.string().max(120).optional().or(z.literal('')),
  linkedin: z.string().url(t('contacts.form.urlInvalid')).optional().or(z.literal('')),
  facebook: z.string().url(t('contacts.form.urlInvalid')).optional().or(z.literal('')),
  instagram: z.string().url(t('contacts.form.urlInvalid')).optional().or(z.literal('')),
  twitter: z.string().url(t('contacts.form.urlInvalid')).optional().or(z.literal('')),
  github: z.string().url(t('contacts.form.urlInvalid')).optional().or(z.literal(''))
}))

const socialFields = [
  { key: 'linkedin' as const, icon: 'i-simple-icons-linkedin' },
  { key: 'facebook' as const, icon: 'i-simple-icons-facebook' },
  { key: 'instagram' as const, icon: 'i-simple-icons-instagram' },
  { key: 'twitter' as const, icon: 'i-simple-icons-x' },
  { key: 'github' as const, icon: 'i-simple-icons-github' }
]

const countryItems = computed(() =>
  countries.map(c => ({
    label: c.name,
    value: c.name,
    emoji: c.emoji
  }))
)

function splitName(full: string | null): { firstName: string, lastName: string } {
  if (!full) return { firstName: '', lastName: '' }
  const parts = full.trim().split(/\s+/)
  if (parts.length === 0) return { firstName: '', lastName: '' }
  if (parts.length === 1) return { firstName: parts[0] ?? '', lastName: '' }
  return {
    firstName: parts[0] ?? '',
    lastName: parts.slice(1).join(' ')
  }
}

function hydrate(): Schema {
  const attrs = parseJsonAttrs(props.contact.additionalAttributes)
  const social = (attrs.social_profiles ?? {}) as Record<string, string>
  const split = splitName(props.contact.name)
  return {
    firstName: split.firstName,
    lastName: split.lastName,
    email: props.contact.email || '',
    phoneNumber: props.contact.phoneNumber || '',
    city: (attrs.city as string) || '',
    country: (attrs.country as string) || '',
    bio: (attrs.description as string) || '',
    companyName: (attrs.company_name as string) || '',
    linkedin: social.linkedin || '',
    facebook: social.facebook || '',
    instagram: social.instagram || '',
    twitter: social.twitter || '',
    github: social.github || ''
  }
}

const state = reactive<Schema>(hydrate())

const form = useTemplateRef('form')

function buildAdditionalAttributes(data: Schema): Record<string, unknown> {
  const socialProfiles: Record<string, string> = {}
  for (const f of socialFields) {
    const v = data[f.key]
    if (v) socialProfiles[f.key] = v
  }

  const result: Record<string, unknown> = {}
  if (data.city) result.city = data.city
  if (data.country) result.country = data.country
  if (data.bio) result.description = data.bio
  if (data.companyName) result.company_name = data.companyName
  if (Object.keys(socialProfiles).length) result.social_profiles = socialProfiles
  return result
}

async function onSubmit(event: FormSubmitEvent<Schema>) {
  const data = event.data
  const fullName = [data.firstName, data.lastName].filter(Boolean).join(' ').trim()

  emit('update', {
    id: props.contact.id,
    name: fullName,
    email: data.email || undefined,
    phone_number: data.phoneNumber || undefined,
    additional_attributes: buildAdditionalAttributes(data)
  })
}
</script>

<template>
  <UForm
    ref="form"
    :schema="schema"
    :state="state"
    class="space-y-4"
    @submit="onSubmit"
  >
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <UFormField name="firstName" required>
        <UInput
          v-model="state.firstName"
          :placeholder="t('contacts.form.firstNamePlaceholder')"
          :disabled="loading"
          icon="i-lucide-user"
          class="w-full"
        />
      </UFormField>
      <UFormField name="lastName">
        <UInput
          v-model="state.lastName"
          :placeholder="t('contacts.form.lastNamePlaceholder')"
          :disabled="loading"
          class="w-full"
        />
      </UFormField>
      <UFormField name="email">
        <UInput
          v-model="state.email"
          type="email"
          :placeholder="t('contacts.form.emailPlaceholder')"
          :disabled="loading"
          icon="i-lucide-mail"
          class="w-full"
        />
      </UFormField>
      <UFormField name="phoneNumber">
        <PhoneNumberInput
          v-model="state.phoneNumber"
          :disabled="loading"
        />
      </UFormField>
      <UFormField name="city">
        <UInput
          v-model="state.city"
          :placeholder="t('contacts.form.cityPlaceholder')"
          :disabled="loading"
          icon="i-lucide-map-pin"
          class="w-full"
        />
      </UFormField>
      <UFormField name="country">
        <USelectMenu
          v-model="state.country"
          :items="countryItems"
          value-key="value"
          :placeholder="t('contacts.form.countryPlaceholder')"
          :search-input="{
            placeholder: t('phoneInput.searchPlaceholder'),
            icon: 'i-lucide-search'
          }"
          :filter-fields="['label']"
          :disabled="loading"
          icon="i-lucide-globe"
          class="w-full"
        >
          <template #item-leading="{ item }">
            <span class="size-5 flex items-center text-lg">{{ item.emoji }}</span>
          </template>
        </USelectMenu>
      </UFormField>
      <UFormField name="companyName" class="md:col-span-2">
        <UInput
          v-model="state.companyName"
          :placeholder="t('contacts.form.companyPlaceholder')"
          :disabled="loading"
          icon="i-lucide-building-2"
          class="w-full"
        />
      </UFormField>
      <UFormField name="bio" class="md:col-span-2">
        <UTextarea
          v-model="state.bio"
          :placeholder="t('contacts.form.bioPlaceholder')"
          :disabled="loading"
          :rows="2"
          autoresize
          class="w-full"
        />
      </UFormField>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <UFormField
        v-for="field in socialFields"
        :key="field.key"
        :name="field.key"
      >
        <UInput
          v-model="state[field.key]"
          :placeholder="t(`contacts.form.${field.key}Placeholder`)"
          :disabled="loading"
          :icon="field.icon"
          type="url"
          class="w-full"
        />
      </UFormField>
    </div>

    <div class="flex items-center justify-end pt-2 border-t border-default/50">
      <UButton
        type="submit"
        :label="t('contacts.card.updateButton')"
        :loading="loading"
        :disabled="loading"
        icon="i-lucide-save"
      />
    </div>
  </UForm>
</template>
