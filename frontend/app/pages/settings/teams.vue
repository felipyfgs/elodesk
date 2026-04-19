<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useTeamsStore, type Team } from '~/stores/teams'
import { teamSchema, type TeamForm } from '~/schemas/team'

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useTeamsStore()

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

const open = ref(false)
const editing = ref<Team | null>(null)
const saved = ref(false)
const errors = ref<Record<string, string>>({})

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
  errors.value = {}
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
  errors.value = {}
  open.value = true
}

async function submit() {
  const result = teamSchema.safeParse(form)
  if (!result.success) {
    errors.value = Object.fromEntries(result.error.issues.map(i => [i.path.join('.'), i.message]))
    return
  }
  errors.value = {}
  loading.value = true
  try {
    if (editing.value) {
      const updated = await api<Team>(`/teams/${editing.value.id}`, { method: 'PUT', body: result.data })
      store.upsert(updated)
    } else {
      const created = await api<Team>('/teams', { method: 'POST', body: result.data })
      store.upsert(created)
    }
    saved.value = true
    open.value = false
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    loading.value = false
  }
}

async function remove(team: Team) {
  if (!confirm(t('teams.deleteConfirm'))) return
  await api(`/teams/${team.id}`, { method: 'DELETE' })
  store.remove(team.id)
}

async function fetchTeams() {
  const list = await api<Team[]>('/teams')
  store.setAll(list)
}

onMounted(fetchTeams)
</script>

<template>
  <div v-if="!isAdmin" class="text-center text-muted py-12">
    {{ t('settings.accessDenied') }}
  </div>

  <template v-else>
    <div v-if="saved" class="mb-4 text-sm text-green-600">
      {{ t('common.success') }}
    </div>

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
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-[var(--ui-border)]"
        >
          <div class="min-w-0">
            <div class="font-medium">{{ team.name }}</div>
            <div v-if="team.description" class="text-sm text-muted truncate">
              {{ team.description }}
            </div>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <UBadge v-if="store.membersByTeam[team.id]" variant="subtle">
              {{ store.membersByTeam[team.id].length }} {{ t('teams.members') }}
            </UBadge>
            <UButton size="xs" variant="ghost" @click="openEdit(team)">
              {{ t('common.edit') }}
            </UButton>
            <UButton size="xs" color="red" variant="ghost" @click="remove(team)">
              {{ t('common.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UPageCard>

    <UModal v-model:open="open" :title="editing ? t('teams.edit') : t('teams.create')">
      <form class="flex flex-col gap-4" @submit.prevent="submit">
        <UFormField :label="t('teams.name')" :error="errors.name">
          <UInput v-model="form.name" class="w-full" />
        </UFormField>

        <UFormField :label="t('teams.description')" :error="errors.description">
          <UTextarea v-model="form.description!" class="w-full" />
        </UFormField>

        <UFormField>
          <UCheckbox v-model="form.allow_auto_assign" :label="t('teams.allowAutoAssign')" />
        </UFormField>

        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="open = false">
            {{ t('common.cancel') }}
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ t('common.save') }}
          </UButton>
        </div>
      </form>
    </UModal>
  </template>
</template>
