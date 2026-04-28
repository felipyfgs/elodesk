<script setup lang="ts">
import { formatTimeAgo } from '@vueuse/core'
import type { Contact } from '~/stores/contacts'
import { parseJsonAttrs } from '~/utils/jsonAttrs'
import ContactsEditForm from './EditForm.vue'

const props = defineProps<{
  contact: Contact
  isExpanded?: boolean
  isSelected?: boolean
  selectable?: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  toggle: []
  select: [value: boolean]
  update: [payload: { id: string, name?: string, email?: string, phone_number?: string, additional_attributes?: Record<string, unknown> }]
  delete: [id: string]
  showDetails: [id: string]
}>()

const { t } = useI18n()

const isUpdating = ref(false)
const confirmDeleteOpen = ref(false)

const initials = computed(() => {
  const name = props.contact.name || '?'
  return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2)
})

const extras = computed(() => parseJsonAttrs(props.contact.additionalAttributes))

const companyName = computed(() => {
  const v = extras.value.company_name ?? extras.value.companyName
  return typeof v === 'string' && v.length > 0 ? v : ''
})

const location = computed(() => {
  const city = typeof extras.value.city === 'string' ? extras.value.city : ''
  const country = typeof extras.value.country === 'string' ? extras.value.country : ''
  return [city, country].filter(Boolean).join(', ')
})

const lastActivity = computed(() => {
  if (!props.contact.lastActivityAt) return ''
  return formatTimeAgo(new Date(props.contact.lastActivityAt))
})

function handleToggle() {
  emit('toggle')
}

function handleSelect(checked: boolean) {
  emit('select', checked)
}

function handleShowDetails() {
  emit('showDetails', props.contact.id)
}

async function handleUpdate(payload: { id: string, name?: string, email?: string, phone_number?: string, additional_attributes?: Record<string, unknown> }) {
  isUpdating.value = true
  try {
    emit('update', payload)
  } finally {
    isUpdating.value = false
  }
}

function handleCancel() {
  emit('toggle')
}

function handleDelete() {
  emit('delete', props.contact.id)
}
</script>

<template>
  <UCard
    :ui="{
      root: [
        'group transition-all duration-200 ring-1 ring-default',
        isSelected ? 'bg-elevated ring-primary/60' : 'bg-default hover:bg-elevated/60'
      ],
      body: 'p-0 sm:p-0'
    }"
  >
    <div
      class="flex items-center gap-4 px-4 py-3 cursor-pointer select-none"
      @click="handleToggle"
    >
      <!-- Avatar com checkbox overlay -->
      <div class="relative flex-shrink-0" @click.stop>
        <UAvatar
          :text="initials"
          :src="contact.avatarUrl || undefined"
          size="lg"
          class="ring-2 ring-background transition-transform group-hover:scale-105"
        />

        <div
          v-if="selectable || isSelected"
          class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full bg-background p-0.5"
        >
          <UCheckbox
            :model-value="isSelected"
            @update:model-value="(val: boolean | 'indeterminate') => val !== 'indeterminate' && handleSelect(val)"
          />
        </div>
      </div>

      <!-- Informações do contato -->
      <div class="flex-1 min-w-0">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-0.5 mb-0.5">
          <span class="text-base font-semibold truncate text-highlighted">
            {{ contact.name || t('contacts.card.noName') }}
          </span>
          <UBadge
            v-if="companyName"
            variant="subtle"
            color="neutral"
            size="sm"
            class="max-w-[150px]"
          >
            <template #leading>
              <UIcon name="i-lucide-building-2" class="size-3" />
            </template>
            <span class="truncate">{{ companyName }}</span>
          </UBadge>
        </div>

        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted">
          <div v-if="contact.email" class="flex items-center gap-1.5 min-w-0" :title="contact.email">
            <UIcon name="i-lucide-mail" class="size-3.5 shrink-0" />
            <span class="truncate">{{ contact.email }}</span>
          </div>
          <div v-if="contact.phoneNumber" class="flex items-center gap-1.5">
            <UIcon name="i-lucide-phone" class="size-3.5 shrink-0" />
            <span class="truncate">{{ contact.phoneNumber }}</span>
          </div>
          <div v-if="location" class="flex items-center gap-1.5 min-w-0">
            <UIcon name="i-lucide-map-pin" class="size-3.5 shrink-0" />
            <span class="truncate">{{ location }}</span>
          </div>

          <div class="flex items-center gap-3 ml-auto">
            <span v-if="lastActivity" class="text-xs text-dimmed italic">
              {{ lastActivity }}
            </span>
            <UButton
              :label="t('contacts.card.viewDetails')"
              variant="link"
              size="sm"
              class="!p-0 font-medium"
              @click.stop="handleShowDetails"
            />
          </div>
        </div>
      </div>

      <div class="flex flex-col items-center gap-1">
        <UButton
          :icon="isExpanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          color="neutral"
          variant="ghost"
          size="sm"
          square
          class="transition-transform duration-200"
        />
      </div>
    </div>

    <!-- Conteúdo expandido -->
    <UCollapsible
      :open="isExpanded"
      class="w-full"
      @update:open="(val: boolean) => val !== isExpanded && handleToggle()"
    >
      <template #content>
        <div class="border-t border-default bg-elevated/20">
          <div class="p-6">
            <ContactsEditForm
              :contact="contact"
              :loading="isUpdating || loading"
              @update="handleUpdate"
              @cancel="handleCancel"
            />
          </div>

          <div class="border-t border-default px-6 py-3 flex items-center justify-end">
            <UPopover v-model:open="confirmDeleteOpen" :content="{ align: 'end', side: 'top' }">
              <UButton
                :label="t('contacts.card.deleteButton')"
                icon="i-lucide-trash"
                color="error"
                variant="ghost"
                size="xs"
              />

              <template #content>
                <div class="p-4 w-72 flex flex-col gap-3">
                  <div class="flex items-start gap-2">
                    <UIcon name="i-lucide-alert-triangle" class="size-4 text-error shrink-0 mt-0.5" />
                    <div class="min-w-0">
                      <h4 class="text-sm font-medium text-highlighted">
                        {{ t('contacts.card.dangerZone') }}
                      </h4>
                      <p class="text-xs text-muted mt-0.5">
                        {{ t('contacts.card.dangerZoneDescription') }}
                      </p>
                    </div>
                  </div>
                  <div class="flex items-center justify-end gap-2">
                    <UButton
                      :label="t('common.cancel')"
                      color="neutral"
                      variant="ghost"
                      size="xs"
                      @click="confirmDeleteOpen = false"
                    />
                    <UButton
                      :label="t('contacts.card.confirmDeleteButton')"
                      color="error"
                      variant="solid"
                      size="xs"
                      icon="i-lucide-trash-2"
                      @click="handleDelete"
                    />
                  </div>
                </div>
              </template>
            </UPopover>
          </div>
        </div>
      </template>
    </UCollapsible>
  </UCard>
</template>
