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
  openHour: number
  openMinute: number
  closeHour: number
  closeMinute: number
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

const defaultSlot = (enabled = true): DaySlot => ({
  enabled,
  openHour: 9,
  openMinute: 0,
  closeHour: 18,
  closeMinute: 0
})

const schedule = reactive<BusinessHoursSettings>(
  Object.fromEntries(days.map(day => [day, defaultSlot(day !== 'saturday' && day !== 'sunday')])) as BusinessHoursSettings
)

const allEnabled = computed(() => days.every(day => schedule[day].enabled))

function getSlot(day: DayName): DaySlot {
  return schedule[day]
}

function clampSlot(slot: DaySlot) {
  slot.openHour = Math.min(23, Math.max(0, Number(slot.openHour) || 0))
  slot.closeHour = Math.min(23, Math.max(0, Number(slot.closeHour) || 0))
  slot.openMinute = Math.min(59, Math.max(0, Number(slot.openMinute) || 0))
  slot.closeMinute = Math.min(59, Math.max(0, Number(slot.closeMinute) || 0))
}

function normalizeSchedule(value: unknown): Partial<BusinessHoursSettings> {
  if (!value || typeof value !== 'object') return {}
  return value as Partial<BusinessHoursSettings>
}

function applySchedule(value: unknown) {
  const saved = normalizeSchedule(value)
  for (const day of days) {
    Object.assign(schedule[day], defaultSlot(day !== 'saturday' && day !== 'sunday'), saved[day] ?? {})
    clampSlot(schedule[day])
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
    for (const day of days) clampSlot(schedule[day])
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
        <p class="mt-1 text-xs text-dimmed">
          {{ t('inboxes.timezone') }}: {{ timezone }}
        </p>
      </div>
      <UButton variant="ghost" size="sm" @click="toggleAll(!allEnabled)">
        {{ allEnabled ? t('inboxes.disableAll') : t('inboxes.enableAll') }}
      </UButton>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-8">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>

    <div v-else class="flex flex-col gap-2">
      <div
        v-for="day in days"
        :key="day"
        class="grid gap-3 rounded-lg px-3 py-3 sm:grid-cols-[160px_1fr_auto_1fr]"
        :class="getSlot(day).enabled ? 'bg-elevated' : 'bg-muted opacity-70'"
      >
        <label class="flex items-center gap-3">
          <UCheckbox v-model="getSlot(day).enabled" />
          <span class="text-sm font-medium">
            {{ t(`inboxes.days.${day}`) }}
          </span>
        </label>

        <template v-if="getSlot(day).enabled">
          <div class="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
            <UInput
              v-model.number="getSlot(day).openHour"
              type="number"
              :min="0"
              :max="23"
              size="sm"
            />
            <span class="text-muted">:</span>
            <UInput
              v-model.number="getSlot(day).openMinute"
              type="number"
              :min="0"
              :max="59"
              size="sm"
            />
          </div>
          <span class="hidden self-center text-muted sm:block">-</span>
          <div class="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
            <UInput
              v-model.number="getSlot(day).closeHour"
              type="number"
              :min="0"
              :max="23"
              size="sm"
            />
            <span class="text-muted">:</span>
            <UInput
              v-model.number="getSlot(day).closeMinute"
              type="number"
              :min="0"
              :max="59"
              size="sm"
            />
          </div>
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
