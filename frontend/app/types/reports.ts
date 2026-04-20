export type Period = 'daily' | 'weekly' | 'monthly'

export interface Range {
  start: Date
  end: Date
}

export interface OverviewReport {
  openCount: number
  resolvedCount: number
  firstResponseAvgMinutes?: number | null
  resolutionAvgMinutes?: number | null
  volumeByDay: Array<{ day: string, total: number }>
  statusBreakdown: Record<string, number>
}

export interface EntityMetric {
  entityId: number
  entityName: string
  total: number
  resolved: number
  open: number
}
