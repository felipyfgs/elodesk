<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { ConfirmModal } from '#components'
import { useAuthStore } from '~/stores/auth'
import { useTeamsStore, type Team } from '~/stores/teams'
import { teamSchema, type TeamForm } from '~/schemas/team'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useTeamsStore()
const confirm = useOverlay().create(ConfirmModal)

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<Team | null>(null)
const saved = ref(false)

const form = reactive<TeamForm>({
  name: '',
  description: null,
  allow_auto_assign: false
})

const loading = ref(false)

function resetForm() {
  form.name = ''
  form.description = null
  form.allow_auto_assign = false
  editing.value = null
}

function openCreate() {
  resetForm()
  open.value = true
}

function openEdit(team: Team) {
  editing.value = team
  form.name = team.name
  form.description = team.description
  form.allow_auto_assign = team.allowAutoAssign
  open.value = true
}

async function submit(event: FormSubmitEvent<TeamForm>) {
  loading.value = true
  try {
    if (editing.value) {
      const updated = await api<Team>(`/accounts/${auth.account?.id}/teams/${editing.value.id}`, { method: 'PATCH', body: event.data })
      store.upsert(updated)
    } else {
      const created = await api<Team>(`/accounts/${auth.account?.id}/teams`, { method: 'POST', body: event.data })
      store.upsert(created)
    }
    saved.value = true
    open.value = false
    resetForm()
    setTimeout(() => {
      saved.value = false
    }, 2000)
  } finally {
    loading.value = false
  }
}

function openDelete(team: Team) {
  confirm.open({
    title: t('common.delete'),
    description: t('teams.deleteConfirm'),
    confirmLabel: t('common.delete'),
    itemName: team.name
  }).then(async (ok) => {
    if (!ok) return
    await api(`/accounts/${auth.account?.id}/teams/${team.id}`, { method: 'DELETE' })
    store.remove(team.id)
  })
}

async function fetchTeams() {
  const list = await api<Team[]>(`/accounts/${auth.account?.id}/teams`)
  store.setAll(list)
}

onMounted(fetchTeams)
</script>

<template>
  <div v-if="!isAdmin" class="text-center text-muted py-12">
    {{ t('settings.accessDenied') }}
  </div>

  <template v-else>
    <UAlert
      v-if="saved"
      class="mb-4"
      color="success"
      variant="subtle"
      icon="i-lucide-check-circle"
      :title="t('common.success')"
    />

    <UPageCard :title="t('teams.title')" variant="subtle">
      <template #header>
        <UButton @click="openCreate">
          {{ t('teams.create') }}
        </UButton>
      </template>

      <p v-if="!store.list.length" class="text-sm text-muted">
        {{ t('teams.empty') }}
      </p>

      <div v-else class="flex flex-col gap-2 mt-2">
        <div
          v-for="team in store.list"
          :key="team.id"
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-default"
        >
          <div class="min-w-0">
            <div class="font-medium">
              {{ team.name }}
            </div>
            <div v-if="team.description" class="text-sm text-muted truncate">
              {{ team.description }}
            </div>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <UBadge v-if="store.membersByTeam[team.id]?.length" variant="subtle">
              {{ store.membersByTeam[team.id]!.length }} {{ t('teams.members') }}
            </UBadge>
            <UButton size="xs" variant="ghost" @click="openEdit(team)">
              {{ t('common.edit') }}
            </UButton>
            <UButton
              size="xs"
              color="error"
              variant="ghost"
              @click="openDelete(team)"
            >
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('teams.edit') : t('teams.create')">
      <UForm
        :schema="teamSchema"
        :state="form"
        class="flex flex-col gap-4"
        @submit="submit"
      >
        <UFormField :label="t('teams.name')" name="name">
          <UInput v-model="form.name" class="w-full" />
        </UFormField>

        <UFormField :label="t('teams.description')" name="description">
          <UTextarea v-model="form.description!" class="w-full" />
        </UFormField>

        <UFormField name="allow_auto_assign">
          <UCheckbox v-model="form.allow_auto_assign" :label="t('teams.allowAutoAssign')" />
        </UFormField>

        <div class="flex justify-end gap-2">
          <UButton type="button" variant="ghost" @click="open = false">
            {{ t('common.cancel') }}
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ t('common.save') }}
          </UButton>
        </div>
      </UForm>
    </UModal>
  </template>
</template>
