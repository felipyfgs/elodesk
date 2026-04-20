import { defineStore } from 'pinia'

export interface CannedResponse {
  id: string
  accountId: string
  shortCode: string
  content: string
  createdAt: string
  updatedAt: string
}

export const useCannedResponsesStore = defineStore('cannedResponses', {
  state: () => ({
    list: [] as CannedResponse[],
    loading: false
  }),
  actions: {
    setAll(list: CannedResponse[]) {
      this.list = list
    },
    upsert(item: CannedResponse) {
      const idx = this.list.findIndex(c => c.id === item.id)
      if (idx >= 0) this.list[idx] = item
      else this.list.push(item)
    },
    remove(id: string) {
      this.list = this.list.filter(c => c.id !== id)
    },
    search(term: string): CannedResponse[] {
      if (!term) return this.list
      const lower = term.toLowerCase()
      return this.list
        .map((item) => {
          let priority = 3
          if (item.shortCode.toLowerCase().startsWith(lower)) priority = 1
          else if (item.shortCode.toLowerCase().includes(lower)) priority = 2
          else if (item.content.toLowerCase().includes(lower)) priority = 3
          return { item, priority }
        })
        .sort((a, b) => a.priority - b.priority || a.item.shortCode.localeCompare(b.item.shortCode))
        .map(e => e.item)
    }
  }
})
