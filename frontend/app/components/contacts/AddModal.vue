<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { createContactCreateSchema, type ContactCreateForm } from '~/schemas/contacts'
import { useContactsStore, type Contact } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import { countries } from '~/utils/countries'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const { t } = useI18n()
const errorHandler = useErrorHandler()
const api = useApi()
const auth = useAuthStore()
const contactsStore = useContactsStore()

const isOpen = computed({
  get: () => props.open,
  set: value => emit('update:open', value)
})

const initialState = (): ContactCreateForm => ({
  first_name: '',
  last_name: '',
  email: '',
  phone_number: '',
  city: '',
  country: '',
  bio: '',
  company_name: '',
  linkedin: '',
  facebook: '',
  instagram: '',
  twitter: '',
  github: ''
})

const state = reactive<ContactCreateForm>(initialState())
const loading = ref(false)
const formRef = useTemplateRef('formRef')

function resetState() {
  Object.assign(state, initialState())
}

watch(isOpen, (open) => {
  if (!open) resetState()
})

const schema = computed(() => createContactCreateSchema(t))

const countryItems = computed(() =>
  countries.map(c => ({
    label: c.name,
    value: c.name,
    emoji: c.emoji
  }))
)

const socialFields = [
  { key: 'linkedin', icon: 'i-simple-icons-linkedin' },
  { key: 'facebook', icon: 'i-simple-icons-facebook' },
  { key: 'instagram', icon: 'i-simple-icons-instagram' },
  { key: 'twitter', icon: 'i-simple-icons-x' },
  { key: 'github', icon: 'i-simple-icons-github' }
] as const

function buildAdditionalAttributes(data: ContactCreateForm) {
  const social: Record<string, string> = {}
  for (const f of socialFields) {
    const v = data[f.key]
    if (v) social[f.key] = v
  }

  const attrs: Record<string, unknown> = {}
  if (data.city) attrs.city = data.city
  if (data.country) attrs.country = data.country
  if (data.bio) attrs.description = data.bio
  if (data.company_name) attrs.company_name = data.company_name
  if (Object.keys(social).length > 0) attrs.social_profiles = social
  return Object.keys(attrs).length > 0 ? attrs : undefined
}

async function onSubmit(event: FormSubmitEvent<ContactCreateForm>) {
  loading.value = true
  try {
    const data = event.data
    const fullName = [data.first_name, data.last_name].filter(Boolean).join(' ').trim()

    const res = await api<Contact>(`/accounts/${auth.account!.id}/contacts`, {
      method: 'POST',
      body: {
        name: fullName,
        email: data.email || null,
        phone_number: data.phone_number || null,
        additional_attributes: buildAdditionalAttributes(data)
      }
    })
    contactsStore.upsert(res)
    errorHandler.success(
      t('contacts.create.success'),
      t('contacts.create.successDescription', { name: fullName })
    )
    isOpen.value = false
  } catch (error) {
    errorHandler.handle(error, {
      title: t('contacts.create.failed'),
      onRetry: () => onSubmit(event)
    })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="t('contacts.new')"
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <UForm
        ref="formRef"
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <UFormField name="first_name" required>
            <UInput
              v-model="state.first_name"
              :placeholder="t('contacts.form.firstNamePlaceholder')"
              icon="i-lucide-user"
              class="w-full"
            />
          </UFormField>
          <UFormField name="last_name">
            <UInput
              v-model="state.last_name"
              :placeholder="t('contacts.form.lastNamePlaceholder')"
              class="w-full"
            />
          </UFormField>
          <UFormField name="email">
            <UInput
              v-model="state.email"
              type="email"
              :placeholder="t('contacts.form.emailPlaceholder')"
              icon="i-lucide-mail"
              class="w-full"
            />
          </UFormField>
          <UFormField name="phone_number">
            <PhoneNumberInput v-model="state.phone_number" />
          </UFormField>
          <UFormField name="city">
            <UInput
              v-model="state.city"
              :placeholder="t('contacts.form.cityPlaceholder')"
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
              icon="i-lucide-globe"
              class="w-full"
            >
              <template #item-leading="{ item }">
                <span class="size-5 flex items-center text-lg">{{ item.emoji }}</span>
              </template>
            </USelectMenu>
          </UFormField>
          <UFormField name="company_name" class="md:col-span-2">
            <UInput
              v-model="state.company_name"
              :placeholder="t('contacts.form.companyPlaceholder')"
              icon="i-lucide-building-2"
              class="w-full"
            />
          </UFormField>
          <UFormField name="bio" class="md:col-span-2">
            <UTextarea
              v-model="state.bio"
              :placeholder="t('contacts.form.bioPlaceholder')"
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
              :icon="field.icon"
              type="url"
              class="w-full"
            />
          </UFormField>
        </div>
      </UForm>
    </template>

    <template #footer>
      <div class="flex justify-end items-center gap-3 w-full">
        <UButton
          :label="t('common.cancel')"
          color="neutral"
          variant="ghost"
          @click="isOpen = false"
        />
        <UButton
          :label="t('contacts.form.save')"
          :loading="loading"
          @click="formRef?.submit()"
        />
      </div>
    </template>
  </UModal>
</template>
