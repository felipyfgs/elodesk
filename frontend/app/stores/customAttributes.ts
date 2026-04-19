import { defineStore } from 'pinia'

export interface CustomAttributeDefinition {
  id: string
  accountId: string
  attributeKey: string
  attributeDisplayName: string
  attributeDisplayType: 'text' | 'number' | 'currency' | 'percent' | 'link' | 'date' | 'list' | 'checkbox'
  attributeModel: 'contact' | 'conversation'
  attributeValues: string | null
  attributeDescription: string | null
  regexPattern: string | null
  defaultValue: string | null
  createdAt: string
  updatedAt: string
}

export const useCustomAttributesStore = defineStore('customAttributes', {
  state: () => ({
    definitions: {
      contact: [] as CustomAttributeDefinition[],
      conversation: [] as CustomAttributeDefinition[]
    },
    loading: false
  }),
  getters: {
    byModel(): (model: string) => CustomAttributeDefinition[] {
      return (model: string) => {
        if (model === 'contact') return this.definitions.contact
        if (model === 'conversation') return this.definitions.conversation
        return []
      }
    },
    contactDefinitions(): CustomAttributeDefinition[] {
      return this.definitions.contact
    },
    conversationDefinitions(): CustomAttributeDefinition[] {
      return this.definitions.conversation
    }
  },
  actions: {
    setAll(list: CustomAttributeDefinition[]) {
      this.definitions.contact = list.filter(d => d.attributeModel === 'contact')
      this.definitions.conversation = list.filter(d => d.attributeModel === 'conversation')
    },
    upsert(def: CustomAttributeDefinition) {
      const arr = def.attributeModel === 'contact' ? this.definitions.contact : this.definitions.conversation
      const idx = arr.findIndex(d => d.id === def.id)
      if (idx >= 0) arr[idx] = def
      else arr.push(def)
    },
    remove(id: string, model: string) {
      if (model === 'contact') {
        this.definitions.contact = this.definitions.contact.filter(d => d.id !== id)
      } else {
        this.definitions.conversation = this.definitions.conversation.filter(d => d.id !== id)
      }
    }
  }
})
