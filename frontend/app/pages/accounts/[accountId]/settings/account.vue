<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { accountSettingsSchema, type AccountSettingsForm } from '~/schemas/settings/account'
import { useAuthStore, type AuthAccount } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

interface AccountSettingsResponse extends AuthAccount {
  createdAt?: string
  updatedAt?: string
}

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()

const canManage = computed(() => (auth.accountUser?.role ?? 0) >= 1)
const loading = ref(false)

const localeOptions = computed(() => [
  { label: 'Português (Brasil)', value: 'pt-BR' },
  { label: 'English', value: 'en' }
])

const timezoneOptions = [
  'America/Sao_Paulo',
  'America/New_York',
  'America/Los_Angeles',
  'America/Mexico_City',
  'Europe/Lisbon',
  'UTC'
]

const state = reactive<Partial<AccountSettingsForm>>({
  name: auth.account?.name ?? '',
  locale: normalizeLocale(auth.account?.locale),
  timezone: String(auth.account?.settings?.timezone ?? 'America/Sao_Paulo')
})

const accountStatus = computed(() => auth.account?.status === 1
  ? {
      label: t('settings.account.suspended'),
      color: 'warning' as const
    }
  : {
      label: t('settings.account.active'),
      color: 'success' as const
    })

function normalizeLocale(locale?: string) {
  return locale === 'en' ? 'en' : 'pt-BR'
}

function applyAccount(account: AccountSettingsResponse) {
  const normalized = { ...account, id: String(account.id), locale: normalizeLocale(account.locale) }
  auth.account = normalized
  auth.accounts = auth.accounts.map(item => item.id === normalized.id ? normalized : item)
  if (!auth.accounts.some(item => item.id === normalized.id)) auth.accounts.unshift(normalized)
  auth.persist()
}

async function load() {
  if (!auth.account?.id) return
  const account = await api<AccountSettingsResponse>(`/accounts/${auth.account.id}`)
  applyAccount(account)
  state.name = account.name
  state.locale = normalizeLocale(account.locale)
  state.timezone = String(account.settings?.timezone ?? 'America/Sao_Paulo')
}

async function onSubmit(event: FormSubmitEvent<AccountSettingsForm>) {
  if (!auth.account?.id || !canManage.value) return
  loading.value = true
  try {
    const settings = {
      ...(auth.account.settings ?? {}),
      timezone: event.data.timezone
    }
    const account = await api<AccountSettingsResponse>(`/accounts/${auth.account.id}`, {
      method: 'PATCH',
      body: {
        name: event.data.name,
        locale: event.data.locale,
        settings
      }
    })
    applyAccount(account)
    toast.add({ title: t('settings.account.saved'), color: 'success', icon: 'i-lucide-check-circle' })
  } catch {
    toast.add({ title: t('settings.account.saveFailed'), color: 'error', icon: 'i-lucide-triangle-alert' })
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <UAlert
      v-if="!canManage"
      color="warning"
      variant="subtle"
      icon="i-lucide-lock"
      :title="t('settings.accessDenied')"
    />

    <UPageCard
      :title="t('settings.account.title')"
      :description="t('settings.account.description')"
      icon="i-lucide-building-2"
      variant="subtle"
    >
      <UForm
        :schema="accountSettingsSchema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField :label="t('settings.account.name')" name="name">
          <UInput v-model="state.name" :disabled="!canManage" class="w-full" />
        </UFormField>

        <UFormField :label="t('settings.account.locale')" name="locale">
          <USelect
            v-model="state.locale"
            :items="localeOptions"
            :disabled="!canManage"
            class="w-full"
          />
        </UFormField>

        <UFormField :label="t('settings.account.timezone')" name="timezone">
          <USelect
            v-model="state.timezone"
            :items="timezoneOptions"
            :disabled="!canManage"
            class="w-full"
          />
        </UFormField>

        <div class="flex justify-end">
          <UButton
            type="submit"
            icon="i-lucide-save"
            :loading="loading"
            :disabled="!canManage"
          >
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UPageCard>

    <UPageCard
      :title="t('settings.account.metadata')"
      icon="i-lucide-info"
      variant="subtle"
    >
      <dl class="text-sm flex flex-col gap-3">
        <div class="flex gap-3">
          <dt class="text-muted w-40 shrink-0">
            {{ t('settings.account.slug') }}
          </dt>
          <dd class="font-medium">
            {{ auth.account?.slug ?? '-' }}
          </dd>
        </div>
        <USeparator />
        <div class="flex gap-3">
          <dt class="text-muted w-40 shrink-0">
            {{ t('settings.account.status') }}
          </dt>
          <dd>
            <UBadge :label="accountStatus.label" :color="accountStatus.color" variant="soft" />
          </dd>
        </div>
      </dl>
    </UPageCard>
  </div>
</template>
