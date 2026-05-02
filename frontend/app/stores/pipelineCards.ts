import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'
import type { LinkedEntityType } from '~/schemas/pipeline'

export interface PipelineCard {
  id: number
  pipelineId: number
  stageId: number
  position: number
  title: string
  description?: string | null
  valueCents?: number | null
  valueCurrency?: string | null
  dueDate?: string | null
  customAttrs: Record<string, unknown>
  linkedEntityType?: LinkedEntityType | null
  linkedEntityId?: number | null
  assigneeUserIds: number[]
  labelIds: number[]
  createdBy?: number | null
  createdAt: string
  updatedAt: string
}

export interface CreateCardPayload {
  stage_id: number
  title: string
  description?: string
  value_cents?: number
  value_currency?: string
  due_date?: string
  custom_attrs?: Record<string, unknown>
  linked_entity_type?: LinkedEntityType
  linked_entity_id?: number
}

export interface UpdateCardPayload {
  title?: string
  description?: string | null
  value_cents?: number | null
  value_currency?: string | null
  due_date?: string | null
  custom_attrs?: Record<string, unknown>
}

interface PipelineCardsBucket {
  byStage: Record<string, PipelineCard[]>
  byId: Record<string, PipelineCard>
  isLoading: boolean
}

function emptyBucket(): PipelineCardsBucket {
  return { byStage: {}, byId: {}, isLoading: false }
}

function sortCards(cards: PipelineCard[]) {
  cards.sort((a, b) => a.position - b.position)
}

export const usePipelineCardsStore = defineStore('pipelineCards', {
  state: () => ({
    byPipeline: {} as Record<string, PipelineCardsBucket>
  }),
  getters: {
    cardsInStage(state) {
      return (pipelineId: number | string, stageId: number | string): PipelineCard[] => {
        const bucket = state.byPipeline[String(pipelineId)]
        if (!bucket) return []
        return bucket.byStage[String(stageId)] ?? []
      }
    },
    cardById(state) {
      return (pipelineId: number | string, cardId: number | string): PipelineCard | undefined => {
        const bucket = state.byPipeline[String(pipelineId)]
        return bucket?.byId[String(cardId)]
      }
    },
    bucketLoading(state) {
      return (pipelineId: number | string): boolean => {
        return state.byPipeline[String(pipelineId)]?.isLoading ?? false
      }
    }
  },
  actions: {
    ensureBucket(pipelineId: number | string): PipelineCardsBucket {
      const key = String(pipelineId)
      if (!this.byPipeline[key]) this.byPipeline[key] = emptyBucket()
      return this.byPipeline[key]
    },
    setAll(pipelineId: number | string, cards: PipelineCard[]) {
      const bucket = this.ensureBucket(pipelineId)
      bucket.byStage = {}
      bucket.byId = {}
      for (const c of cards) {
        bucket.byId[String(c.id)] = c
        const stageKey = String(c.stageId)
        if (!bucket.byStage[stageKey]) bucket.byStage[stageKey] = []
        bucket.byStage[stageKey].push(c)
      }
      for (const stageId of Object.keys(bucket.byStage)) {
        sortCards(bucket.byStage[stageId]!)
      }
    },
    upsert(card: PipelineCard) {
      const bucket = this.ensureBucket(card.pipelineId)
      const prev = bucket.byId[String(card.id)]
      const merged: PipelineCard = prev
        ? {
            ...prev,
            ...card,
            customAttrs: card.customAttrs ?? prev.customAttrs,
            assigneeUserIds: card.assigneeUserIds ?? prev.assigneeUserIds,
            labelIds: card.labelIds ?? prev.labelIds
          }
        : card
      bucket.byId[String(card.id)] = merged

      if (prev && String(prev.stageId) !== String(merged.stageId)) {
        const fromKey = String(prev.stageId)
        bucket.byStage[fromKey] = (bucket.byStage[fromKey] ?? []).filter(
          c => String(c.id) !== String(merged.id)
        )
      }
      const stageKey = String(merged.stageId)
      const list = bucket.byStage[stageKey] ?? []
      const idx = list.findIndex(c => String(c.id) === String(merged.id))
      if (idx >= 0) list[idx] = merged
      else list.push(merged)
      bucket.byStage[stageKey] = list
      sortCards(list)
    },
    setMovedCard(payload: {
      cardId: number | string
      pipelineId: number | string
      fromStageId: number | string
      toStageId: number | string
      position: number
    }) {
      const bucket = this.byPipeline[String(payload.pipelineId)]
      if (!bucket) return
      const card = bucket.byId[String(payload.cardId)]
      if (!card) return
      const fromKey = String(payload.fromStageId)
      const toKey = String(payload.toStageId)
      bucket.byStage[fromKey] = (bucket.byStage[fromKey] ?? []).filter(
        c => String(c.id) !== String(payload.cardId)
      )
      const updated: PipelineCard = { ...card, stageId: Number(payload.toStageId), position: payload.position }
      bucket.byId[String(payload.cardId)] = updated
      const list = bucket.byStage[toKey] ?? []
      const idx = list.findIndex(c => String(c.id) === String(payload.cardId))
      if (idx >= 0) list[idx] = updated
      else list.push(updated)
      bucket.byStage[toKey] = list
      sortCards(list)
    },
    remove(pipelineId: number | string, cardId: number | string) {
      const bucket = this.byPipeline[String(pipelineId)]
      if (!bucket) return
      const card = bucket.byId[String(cardId)]
      if (!card) return
      const newById: Record<string, PipelineCard> = {}
      for (const [key, value] of Object.entries(bucket.byId)) {
        if (key !== String(cardId)) newById[key] = value
      }
      bucket.byId = newById
      const stageKey = String(card.stageId)
      bucket.byStage[stageKey] = (bucket.byStage[stageKey] ?? []).filter(
        c => String(c.id) !== String(cardId)
      )
    },
    async fetchCard(cardId: number | string): Promise<PipelineCard | null> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return null
      const res = await api<PipelineCard>(`/accounts/${auth.account.id}/cards/${cardId}`)
      this.upsert(res)
      return res
    },
    async fetchByPipeline(pipelineId: number | string): Promise<PipelineCard[]> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return []
      const bucket = this.ensureBucket(pipelineId)
      bucket.isLoading = true
      try {
        const res = await api<PipelineCard[]>(`/accounts/${auth.account.id}/pipelines/${pipelineId}/cards`)
        const cards = Array.isArray(res) ? res : []
        this.setAll(pipelineId, cards)
        return cards
      } finally {
        bucket.isLoading = false
      }
    },
    async create(pipelineId: number | string, payload: CreateCardPayload): Promise<PipelineCard> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<PipelineCard>(`/accounts/${auth.account.id}/pipelines/${pipelineId}/cards`, {
        method: 'POST',
        body: payload
      })
      this.upsert(res)
      return res
    },
    async update(pipelineId: number | string, cardId: number | string, payload: UpdateCardPayload): Promise<PipelineCard> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<PipelineCard>(`/accounts/${auth.account.id}/cards/${cardId}`, {
        method: 'PATCH',
        body: payload
      })
      this.upsert(res)
      return res
    },
    async move(pipelineId: number | string, cardId: number | string, toStageId: number | string, position: number): Promise<PipelineCard | null> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return null
      const bucket = this.byPipeline[String(pipelineId)]
      const original = bucket?.byId[String(cardId)]
      const originalStageId = original?.stageId
      const originalPosition = original?.position
      if (original) {
        this.setMovedCard({
          cardId,
          pipelineId,
          fromStageId: original.stageId,
          toStageId,
          position
        })
      }
      try {
        const res = await api<PipelineCard>(`/accounts/${auth.account.id}/cards/${cardId}/move`, {
          method: 'POST',
          body: { stage_id: Number(toStageId), position }
        })
        this.upsert(res)
        return res
      } catch (err) {
        if (original && originalStageId !== undefined && originalPosition !== undefined) {
          this.setMovedCard({
            cardId,
            pipelineId,
            fromStageId: toStageId,
            toStageId: originalStageId,
            position: originalPosition
          })
        }
        throw err
      }
    },
    async delete(pipelineId: number | string, cardId: number | string) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/cards/${cardId}`, { method: 'DELETE' })
      this.remove(pipelineId, cardId)
    },
    async addAssignee(pipelineId: number | string, cardId: number | string, userId: number): Promise<PipelineCard> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<PipelineCard>(`/accounts/${auth.account.id}/cards/${cardId}/assignees`, {
        method: 'POST',
        body: { user_id: userId }
      })
      this.upsert(res)
      return res
    },
    async removeAssignee(pipelineId: number | string, cardId: number | string, userId: number): Promise<void> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/cards/${cardId}/assignees/${userId}`, { method: 'DELETE' })
      const card = this.byPipeline[String(pipelineId)]?.byId[String(cardId)]
      if (card) {
        this.upsert({ ...card, assigneeUserIds: card.assigneeUserIds.filter(id => id !== userId) })
      }
    },
    async applyLabel(pipelineId: number | string, cardId: number | string, labelId: number): Promise<PipelineCard> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<PipelineCard>(`/accounts/${auth.account.id}/cards/${cardId}/labels`, {
        method: 'POST',
        body: { label_id: labelId }
      })
      this.upsert(res)
      return res
    },
    async removeLabel(pipelineId: number | string, cardId: number | string, labelId: number): Promise<void> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/cards/${cardId}/labels/${labelId}`, { method: 'DELETE' })
      const card = this.byPipeline[String(pipelineId)]?.byId[String(cardId)]
      if (card) {
        this.upsert({ ...card, labelIds: card.labelIds.filter(id => id !== labelId) })
      }
    },
    applyRealtimeMove(payload: {
      cardId: number | string
      pipelineId: number | string
      fromStageId: number | string
      toStageId: number | string
      position: number
    }) {
      this.setMovedCard(payload)
    }
  }
})
