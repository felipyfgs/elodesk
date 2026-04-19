import { defineStore } from 'pinia'

export interface Note {
  id: string
  accountId: string
  contactId: string
  userId: string
  content: string
  createdAt: string
  updatedAt: string
}

export const useNotesStore = defineStore('notes', {
  state: () => ({
    byContact: {} as Record<string, Note[]>,
    loading: false
  }),
  actions: {
    setForContact(contactId: string, list: Note[]) {
      this.byContact[contactId] = list
    },
    upsert(note: Note) {
      const bucket = (this.byContact[note.contactId] ||= [])
      const idx = bucket.findIndex(n => n.id === note.id)
      if (idx >= 0) bucket[idx] = note
      else bucket.unshift(note)
      bucket.sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    },
    remove(noteId: string, contactId: string) {
      if (this.byContact[contactId]) {
        this.byContact[contactId] = this.byContact[contactId].filter(n => n.id !== noteId)
      }
    }
  }
})
