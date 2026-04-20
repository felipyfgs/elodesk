<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'
import { useNotesStore, type Note } from '~/stores/notes'
import { noteSchema, type NoteForm } from '~/schemas/note'
import { format } from 'date-fns'

const props = defineProps<{
  contactId: string
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const store = useNotesStore()

const notes = computed(() => store.byContact[props.contactId] ?? [])

const createForm = reactive<NoteForm>({
  content: ''
})
const creating = ref(false)
const editingId = ref<string | null>(null)
const editForm = reactive<NoteForm>({
  content: ''
})
const updating = ref(false)
const removeTarget = ref<Note | null>(null)
const removing = ref(false)

async function loadNotes() {
  if (!auth.account?.id) return
  const res = await api<Note[]>(`/accounts/${auth.account.id}/contacts/${props.contactId}/notes`)
  store.setForContact(props.contactId, res)
}

async function createNote(event: FormSubmitEvent<NoteForm>) {
  if (!auth.account?.id) return
  creating.value = true
  try {
    const note = await api<Note>(
      `/accounts/${auth.account.id}/contacts/${props.contactId}/notes`,
      { method: 'POST', body: event.data }
    )
    store.upsert(note)
    createForm.content = ''
  } finally {
    creating.value = false
  }
}

function startEdit(note: Note) {
  editingId.value = note.id
  editForm.content = note.content
}

function cancelEdit() {
  editingId.value = null
  editForm.content = ''
}

async function updateNote(note: Note, event: FormSubmitEvent<NoteForm>) {
  if (!auth.account?.id) return
  updating.value = true
  try {
    const updated = await api<Note>(
      `/accounts/${auth.account.id}/contacts/${props.contactId}/notes/${note.id}`,
      { method: 'PATCH', body: event.data }
    )
    store.upsert(updated)
    cancelEdit()
  } finally {
    updating.value = false
  }
}

function askDelete(note: Note) {
  removeTarget.value = note
}

function _closeDelete() {
  removeTarget.value = null
}

function _onDeleteModalUpdate(value: boolean) {
  if (!value) _closeDelete()
}

async function _deleteNote() {
  if (!removeTarget.value || !auth.account?.id) return
  removing.value = true
  try {
    await api(`/accounts/${auth.account.id}/contacts/${props.contactId}/notes/${removeTarget.value.id}`, {
      method: 'DELETE'
    })
    store.remove(removeTarget.value.id, props.contactId)
    removeTarget.value = null
  } finally {
    removing.value = false
  }
}

const isAdmin = computed(() => (auth.accountUser?.role ?? 0) >= 1)

function canEdit(note: Note) {
  return isAdmin.value || note.userId === auth.user?.id
}

onMounted(loadNotes)
</script>

<template>
  <UPageCard variant="outline" :title="t('notes.title')">
    <UForm
      :schema="noteSchema"
      :state="createForm"
      class="mb-4 flex flex-col gap-2"
      @submit="createNote"
    >
      <UFormField name="content">
        <UTextarea
          v-model="createForm.content"
          :placeholder="t('notes.placeholder')"
          :rows="3"
          :disabled="creating"
          class="w-full"
        />
      </UFormField>
      <div class="flex justify-end">
        <UButton
          type="submit"
          :loading="creating"
          :label="t('notes.create')"
          icon="i-lucide-plus"
        />
      </div>
    </UForm>

    <USeparator v-if="notes.length" class="mb-4" />

    <p v-if="!notes.length" class="text-sm text-muted">
      {{ t('notes.empty') }}
    </p>

    <div v-else class="space-y-3">
      <div
        v-for="note in notes"
        :key="note.id"
        class="rounded-lg border border-default p-3"
      >
        <UForm
          v-if="editingId === note.id"
          :schema="noteSchema"
          :state="editForm"
          class="space-y-2"
          @submit="updateNote(note, $event)"
        >
          <UFormField name="content">
            <UTextarea
              v-model="editForm.content"
              :rows="3"
              :disabled="updating"
              class="w-full"
            />
          </UFormField>
          <div class="flex justify-end gap-2">
            <UButton
              size="xs"
              type="button"
              variant="ghost"
              @click="cancelEdit"
            >
              {{ t('common.cancel') }}
            </UButton>
            <UButton size="xs" type="submit" :loading="updating">
              {{ t('common.save') }}
            </UButton>
          </div>
        </UForm>

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
              <UButton
                size="xs"
                color="error"
                variant="ghost"
                @click="askDelete(note)"
              >
                {{ t('notes.delete') }}
              </UButton>
            </div>
          </div>
        </template>
      </div>
    </div>
  </UPageCard>
</template>
