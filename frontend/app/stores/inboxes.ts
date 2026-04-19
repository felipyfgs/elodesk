import { defineStore } from 'pinia'

export interface Inbox {
  id: string
  name: string
  channelType: string
  channelId: string
  channelApi?: {
    identifier: string
    webhookUrl: string
    hmacMandatory: boolean
  } | null
}

export const useInboxesStore = defineStore('inboxes', {
  state: () => ({
    list: [] as Inbox[],
    loading: false
  }),
  actions: {
    setAll(list: Inbox[]) {
      this.list = list
    },
    upsert(inbox: Inbox) {
      const idx = this.list.findIndex(i => i.id === inbox.id)
      if (idx >= 0) this.list[idx] = inbox
      else this.list.push(inbox)
    }
  }
})
