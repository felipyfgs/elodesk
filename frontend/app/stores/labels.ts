import { defineStore } from 'pinia'

export interface Label {
  id: string
  accountId: string
  title: string
  color: string
  description: string | null
  showOnSidebar: boolean
  createdAt: string
  updatedAt: string
}

export const useLabelsStore = defineStore('labels', {
  state: () => ({
    list: [] as Label[],
    loading: false
  }),
  getters: {
    byId(): (id: string) => Label | undefined {
      return (id: string) => this.list.find(l => l.id === id)
    }
  },
  actions: {
    setAll(list: Label[]) {
      this.list = list
    },
    upsert(label: Label) {
      const idx = this.list.findIndex(l => l.id === label.id)
      if (idx >= 0) this.list[idx] = label
      else this.list.push(label)
    },
    remove(id: string) {
      this.list = this.list.filter(l => l.id !== id)
    },
    removeByIds(ids: string[]) {
      const set = new Set(ids)
      this.list = this.list.filter(l => !set.has(l.id))
    }
  }
})
