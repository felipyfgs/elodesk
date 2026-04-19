<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useNotesStore, type Note } from '~/stores/notes'
import { noteSchema } from '~/schemas/note'
import { format } from 'date-fns'

const props = defineProps<{
  contactId: string
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useNotesStore()

const notes = computed(() => store.byContact[props.contactId] ?? [])

const newContent = ref('')
const creating = ref(false)
const editingId = ref<string | null>(null)
const editContent = ref('')
const updating = ref(false)

async function loadNotes() {
  if (!auth.account?.id) return
  const res = await api<Note[]>(`/accounts/${auth.account.id}/contacts/${props.contactId}/notes`)
  store.setForContact(props.contactId, res)
}

async function createNote() {
  const result = noteSchema.safeParse({ content: newContent.value })
  if (!result.success) return

  creating.value = true
  try {
    const note = await api<Note>(
      `/accounts/${auth.account.id}/contacts/${props.contactId}/notes`,
      { method: 'POST', body: result.data }
    )
    store.upsert(note)
    newContent.value = ''
  } finally {
    creating.value = false
  }
}

function startEdit(note: Note) {
  editingId.value = note.id
  editContent.value = note.content
}

function cancelEdit() {
  editingId.value = null
  editContent.value = ''
}

async function updateNote(note: Note) {
  const result = noteSchema.safeParse({ content: editContent.value })
  if (!result.success) return

  updating.value = true
  try {
    const updated = await api<Note>(
      `/accounts/${auth.account.id}/contacts/${props.contactId}/notes/${note.id}`,
      { method: 'PATCH', body: result.data }
    )
    store.upsert(updated)
    editingId.value = null
  } finally {
    updating.value = false
  }
}

async function deleteNote(note: Note) {
  if (!confirm(t('notes.delete'))) return
  await api(`/accounts/${auth.account.id}/contacts/${props.contactId}/notes/${note.id}`, {
    method: 'DELETE'
  })
  store.remove(note.id, props.contactId)
}

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

function canEdit(note: Note) {
  return isAdmin.value || note.userId === auth.user?.id
}

onMounted(loadNotes)
</script>

<template>
  <UPageCard variant="outline" :title="t('notes.title')">
    <form class="mb-4 flex flex-col gap-2" @submit.prevent="createNote">
      <UTextarea
        v-model="newContent"
        :placeholder="t('notes.placeholder')"
        :rows="3"
        :disabled="creating"
        class="w-full"
      />
      <div class="flex justify-end">
        <UButton type="submit" :loading="creating" :label="t('notes.create')" icon="i-lucide-plus" />
      </div>
    </form>

    <USeparator v-if="notes.length" class="mb-4" />

    <p v-if="!notes.length" class="text-sm text-muted">
      {{ t('notes.empty') }}
    </p>

    <div v-else class="space-y-3">
      <div
        v-for="note in notes"
        :key="note.id"
        class="rounded-lg border border-[var(--ui-border)] p-3"
      >
        <div v-if="editingId === note.id" class="space-y-2">
          <UTextarea v-model="editContent" :rows="3" :disabled="updating" class="w-full" />
          <div class="flex justify-end gap-2">
            <UButton size="xs" variant="ghost" @click="cancelEdit">
              {{ t('common.cancel') }}
            </UButton>
            <UButton size="xs" :loading="updating" @click="updateNote(note)">
              {{ t('common.save') }}
            </UButton>
          </div>
        </div>

        <template v-else>
          <p class="text-sm whitespace-pre-wrap">
            {{ note.content }}
          </p>
          <div class="flex items-center justify-between mt-2">
            <span class="text-xs text-muted">
              {{ format(new Date(note.createdAt), 'MMM d, yyyy HH:mm') }}
            </span>
            <div v-if="canEdit(note)" class="flex gap-1">
              <UButton size="xs" variant="ghost" @click="startEdit(note)">
                {{ t('notes.edit') }}
              </UButton>
              <UButton size="xs" color="red" variant="ghost" @click="deleteNote(note)">
                {{ t('notes.delete') }}
              </UButton>
            </div>
          </div>
        </template>
      </div>
    </div>
  </UPageCard>
</template>
