import { defineStore } from 'pinia'

export interface SavedFilter {
  id: string
  accountId: string
  userId: string
  name: string
  filterType: 'conversation' | 'contact'
  query: string | null
  createdAt: string
  updatedAt: string
}

export const useSavedFiltersStore = defineStore('savedFilters', {
  state: () => ({
    list: [] as SavedFilter[],
    loading: false
  }),
  getters: {
    byType(): (filterType: string) => SavedFilter[] {
      return (filterType: string) => this.list.filter(f => f.filterType === filterType)
    },
    conversationFilters(): SavedFilter[] {
      return this.list.filter(f => f.filterType === 'conversation')
    },
    contactFilters(): SavedFilter[] {
      return this.list.filter(f => f.filterType === 'contact')
    }
  },
  actions: {
    setAll(list: SavedFilter[]) {
      this.list = list
    },
    upsert(filter: SavedFilter) {
      const idx = this.list.findIndex(f => f.id === filter.id)
      if (idx >= 0) this.list[idx] = filter
      else this.list.push(filter)
    },
    remove(id: string) {
      this.list = this.list.filter(f => f.id !== id)
    }
  }
})
