<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Inbox } from '~/stores/inboxes'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const days = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'] as const
type DayName = (typeof days)[number]

interface DaySlot {
  enabled: boolean
  openTime: string
  closeTime: string
}

type BusinessHoursSettings = Record<DayName, DaySlot>

interface BusinessHoursResponse {
  inboxId: number
  timezone: string
  schedule: BusinessHoursSettings
  createdAt?: string
  updatedAt?: string
}

const loading = ref(true)
const saving = ref(false)
const timezone = ref(String(auth.account?.settings?.timezone ?? 'America/Sao_Paulo'))

const timezones = [
  { label: 'America/Sao_Paulo (GMT-03:00)', value: 'America/Sao_Paulo' },
  { label: 'America/New_York (GMT-05:00)', value: 'America/New_York' },
  { label: 'America/Chicago (GMT-06:00)', value: 'America/Chicago' },
  { label: 'America/Denver (GMT-07:00)', value: 'America/Denver' },
  { label: 'America/Los_Angeles (GMT-08:00)', value: 'America/Los_Angeles' },
  { label: 'Europe/London (GMT+00:00)', value: 'Europe/London' },
  { label: 'Europe/Paris (GMT+01:00)', value: 'Europe/Paris' },
  { label: 'Europe/Berlin (GMT+01:00)', value: 'Europe/Berlin' },
  { label: 'Asia/Tokyo (GMT+09:00)', value: 'Asia/Tokyo' },
  { label: 'Asia/Shanghai (GMT+08:00)', value: 'Asia/Shanghai' },
  { label: 'Asia/Kolkata (GMT+05:30)', value: 'Asia/Kolkata' },
  { label: 'Australia/Sydney (GMT+11:00)', value: 'Australia/Sydney' },
  { label: 'Pacific/Auckland (GMT+13:00)', value: 'Pacific/Auckland' }
]

const defaultSlot = (enabled = true): DaySlot => ({
  enabled,
  openTime: '09:00',
  closeTime: '18:00'
})

const schedule = reactive<BusinessHoursSettings>(
  Object.fromEntries(days.map(day => [day, defaultSlot(day !== 'saturday' && day !== 'sunday')])) as BusinessHoursSettings
)

const allEnabled = computed(() => days.every(day => schedule[day].enabled))

function getSlot(day: DayName): DaySlot {
  return schedule[day]
}

function normalizeSchedule(value: unknown): Partial<BusinessHoursSettings> {
  if (!value || typeof value !== 'object') return {}
  return value as Partial<BusinessHoursSettings>
}

function applySchedule(value: unknown) {
  const saved = normalizeSchedule(value)
  for (const day of days) {
    const s = saved[day]
    if (s) {
      schedule[day].enabled = s.enabled ?? (day !== 'saturday' && day !== 'sunday')
      schedule[day].openTime = s.openTime || '09:00'
      schedule[day].closeTime = s.closeTime || '18:00'
    } else {
      Object.assign(schedule[day], defaultSlot(day !== 'saturday' && day !== 'sunday'))
    }
  }
}

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    const res = await api<BusinessHoursResponse>(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}/business_hours`)
    timezone.value = res.timezone || timezone.value
    applySchedule(res.schedule)
  } finally {
    loading.value = false
  }
}

function toggleAll(enabled: boolean) {
  for (const day of days) {
    schedule[day].enabled = enabled
  }
}

async function save() {
  if (!auth.account?.id) return
  saving.value = true
  try {
    const res = await api<BusinessHoursResponse>(`/accounts/${auth.account.id}/inboxes/${props.inbox.id}/business_hours`, {
      method: 'PUT',
      body: {
        timezone: timezone.value,
        schedule: JSON.parse(JSON.stringify(schedule))
      }
    })
    timezone.value = res.timezone || timezone.value
    applySchedule(res.schedule)
    toast.add({ title: t('common.success'), color: 'success' })
  } catch (err: unknown) {
    const e = err as { data?: { message?: string } }
    toast.add({ title: e?.data?.message || t('common.error'), color: 'error' })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p class="text-sm text-muted">
          {{ t('inboxes.businessHoursDescription') }}
        </p>
      </div>
      <UButton variant="ghost" size="sm" @click="toggleAll(!allEnabled)">
        {{ allEnabled ? t('inboxes.disableAll') : t('inboxes.enableAll') }}
      </UButton>
    </div>

    <UFormField :label="t('inboxes.timezone')">
      <USelect
        v-model="timezone"
        :items="timezones"
        value-key="value"
        label-key="label"
        class="w-full"
      />
    </UFormField>

    <div v-if="loading" class="flex flex-col gap-2">
      <USkeleton v-for="n in 7" :key="n" class="h-14 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-2">
      <div
        v-for="day in days"
        :key="day"
        class="grid gap-3 rounded-lg px-3 py-3 sm:grid-cols-[160px_1fr_auto_1fr]"
        :class="getSlot(day).enabled ? 'bg-elevated' : 'bg-muted opacity-70'"
      >
        <div class="flex items-center gap-3">
          <USwitch v-model="getSlot(day).enabled" />
          <span class="text-sm font-medium">
            {{ t(`inboxes.days.${day}`) }}
          </span>
        </div>

        <template v-if="getSlot(day).enabled">
          <UInput
            v-model="getSlot(day).openTime"
            type="time"
            size="sm"
          />
          <span class="hidden self-center text-muted sm:block">-</span>
          <UInput
            v-model="getSlot(day).closeTime"
            type="time"
            size="sm"
          />
        </template>

        <span v-else class="text-sm text-muted sm:col-span-3">
          {{ t('inboxes.closed') }}
        </span>
      </div>
    </div>

    <div class="flex justify-end">
      <UButton :loading="saving" @click="save">
        {{ t('common.save') }}
      </UButton>
    </div>
  </div>
</template>
