import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'
import type { CardKind, TerminalKind } from '~/schemas/pipeline'

export interface PipelineStage {
  id: number
  pipelineId: number
  name: string
  position: number
  color: string
  isTerminal: boolean
  terminalKind?: TerminalKind | null
  createdAt: string
  updatedAt: string
}

export interface Pipeline {
  id: number
  accountId: number
  name: string
  description?: string | null
  templateKey?: string | null
  cardKind: CardKind
  icon?: string | null
  color: string
  archivedAt?: string | null
  createdBy?: number | null
  createdAt: string
  updatedAt: string
  stages?: PipelineStage[]
}

export interface PipelineTemplateStage {
  name: string
  color: string
  isTerminal?: boolean
  terminalKind?: TerminalKind
}

export interface PipelineTemplate {
  key: string
  name: string
  description: string
  icon: string
  color: string
  cardKind: CardKind
  stages: PipelineTemplateStage[]
}

export interface CreatePipelinePayload {
  name: string
  description?: string
  template_key?: string
  card_kind?: CardKind
  icon?: string
  color?: string
  stages?: { name: string, color: string, is_terminal?: boolean, terminal_kind?: TerminalKind }[]
}

export interface UpdatePipelinePayload {
  name?: string
  description?: string | null
  icon?: string
  color?: string
}

export interface CreateStagePayload {
  name: string
  color?: string
  is_terminal?: boolean
  terminal_kind?: TerminalKind
}

export interface UpdateStagePayload {
  name?: string
  color?: string
  position?: number
  is_terminal?: boolean
  terminal_kind?: TerminalKind
}

export const usePipelinesStore = defineStore('pipelines', {
  state: () => ({
    list: [] as Pipeline[],
    templates: [] as PipelineTemplate[],
    isLoading: false
  }),
  getters: {
    byId(state): (id: number | string) => Pipeline | undefined {
      return (id: number | string) => state.list.find(p => String(p.id) === String(id))
    },
    activeList(state): Pipeline[] {
      return state.list.filter(p => !p.archivedAt)
    }
  },
  actions: {
    setAll(items: Pipeline[]) {
      this.list = Array.isArray(items) ? items : []
    },
    upsert(p: Pipeline) {
      const idx = this.list.findIndex(item => String(item.id) === String(p.id))
      if (idx >= 0) this.list[idx] = { ...this.list[idx]!, ...p }
      else this.list.push(p)
    },
    remove(id: number | string) {
      this.list = this.list.filter(p => String(p.id) !== String(id))
    },
    upsertStage(stage: PipelineStage) {
      const pipeline = this.byId(stage.pipelineId)
      if (!pipeline) return
      const stages = pipeline.stages ?? []
      const idx = stages.findIndex(s => String(s.id) === String(stage.id))
      if (idx >= 0) stages[idx] = { ...stages[idx]!, ...stage }
      else stages.push(stage)
      stages.sort((a, b) => a.position - b.position)
      pipeline.stages = stages
    },
    removeStage(pipelineId: number | string, stageId: number | string) {
      const pipeline = this.byId(pipelineId)
      if (!pipeline?.stages) return
      pipeline.stages = pipeline.stages.filter(s => String(s.id) !== String(stageId))
    },
    async fetchAll(includeArchived = false) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      this.isLoading = true
      try {
        const url = includeArchived
          ? `/accounts/${auth.account.id}/pipelines?archived=true`
          : `/accounts/${auth.account.id}/pipelines`
        const res = await api<Pipeline[]>(url)
        this.setAll(res)
      } finally {
        this.isLoading = false
      }
    },
    async fetchOne(id: number | string): Promise<Pipeline | null> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return null
      const res = await api<Pipeline>(`/accounts/${auth.account.id}/pipelines/${id}`)
      this.upsert(res)
      return res
    },
    async fetchTemplates() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const res = await api<PipelineTemplate[]>(`/accounts/${auth.account.id}/pipelines/templates`)
      this.templates = Array.isArray(res) ? res : []
    },
    async create(payload: CreatePipelinePayload): Promise<Pipeline> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<Pipeline>(`/accounts/${auth.account.id}/pipelines`, {
        method: 'POST',
        body: payload
      })
      this.upsert(res)
      return res
    },
    async update(id: number | string, payload: UpdatePipelinePayload): Promise<Pipeline> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<Pipeline>(`/accounts/${auth.account.id}/pipelines/${id}`, {
        method: 'PATCH',
        body: payload
      })
      this.upsert(res)
      return res
    },
    async archive(id: number | string) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/pipelines/${id}`, { method: 'DELETE' })
      const p = this.byId(id)
      if (p) {
        p.archivedAt = new Date().toISOString()
        this.upsert(p)
      }
    },
    async addStage(pipelineId: number | string, payload: CreateStagePayload): Promise<PipelineStage> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<PipelineStage>(`/accounts/${auth.account.id}/pipelines/${pipelineId}/stages`, {
        method: 'POST',
        body: payload
      })
      this.upsertStage(res)
      return res
    },
    async updateStage(pipelineId: number | string, stageId: number | string, payload: UpdateStagePayload): Promise<PipelineStage> {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No active account')
      const res = await api<PipelineStage>(`/accounts/${auth.account.id}/pipelines/${pipelineId}/stages/${stageId}`, {
        method: 'PATCH',
        body: payload
      })
      this.upsertStage(res)
      return res
    },
    async deleteStage(pipelineId: number | string, stageId: number | string) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/pipelines/${pipelineId}/stages/${stageId}`, { method: 'DELETE' })
      this.removeStage(pipelineId, stageId)
    }
  }
})
