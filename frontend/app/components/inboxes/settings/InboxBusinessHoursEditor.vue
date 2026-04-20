<script setup lang="ts">
const { t } = useI18n()

const days = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'] as const
type DayName = (typeof days)[number]

interface DaySlot {
  enabled: boolean
  openHour: number
  openMinute: number
  closeHour: number
  closeMinute: number
}

const defaultSlot = (): DaySlot => ({
  enabled: true,
  openHour: 9,
  openMinute: 0,
  closeHour: 18,
  closeMinute: 0
})

const schedule = reactive<Record<DayName, DaySlot>>(
  Object.fromEntries(days.map(d => [d, { ...defaultSlot(), enabled: d !== 'saturday' && d !== 'sunday' }])) as Record<DayName, DaySlot>
)

function getSlot(day: DayName): DaySlot {
  return schedule[day]
}

function toggleAll(enabled: boolean) {
  for (const day of days) {
    schedule[day].enabled = enabled
  }
}

const allEnabled = computed(() => days.every(d => schedule[d].enabled))
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted">
        {{ t('inboxes.businessHoursDescription') }}
      </p>
      <UButton
        variant="ghost"
        size="sm"
        @click="toggleAll(!allEnabled)"
      >
        {{ allEnabled ? t('inboxes.disableAll') : t('inboxes.enableAll') }}
      </UButton>
    </div>

    <div class="flex flex-col gap-2">
      <div
        v-for="day in days"
        :key="day"
        class="flex items-center gap-3 rounded-lg px-3 py-2"
        :class="getSlot(day).enabled ? 'bg-[var(--ui-bg-accented)]' : 'opacity-50'"
      >
        <UCheckbox v-model="getSlot(day).enabled" />
        <span class="text-sm font-medium w-24">
          {{ t(`inboxes.days.${day}`) }}
        </span>
        <template v-if="getSlot(day).enabled">
          <UInput
            v-model.number="getSlot(day).openHour"
            type="number"
            :min="0"
            :max="23"
            class="w-20"
            size="sm"
          />
          <span class="text-muted">:</span>
          <UInput
            v-model.number="getSlot(day).openMinute"
            type="number"
            :min="0"
            :max="59"
            class="w-20"
            size="sm"
          />
          <span class="text-muted mx-1">—</span>
          <UInput
            v-model.number="getSlot(day).closeHour"
            type="number"
            :min="0"
            :max="23"
            class="w-20"
            size="sm"
          />
          <span class="text-muted">:</span>
          <UInput
            v-model.number="getSlot(day).closeMinute"
            type="number"
            :min="0"
            :max="59"
            class="w-20"
            size="sm"
          />
        </template>
        <span v-else class="text-sm text-muted">
          {{ t('inboxes.closed') }}
        </span>
      </div>
    </div>
  </div>
</template>
