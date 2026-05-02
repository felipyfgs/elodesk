import type { PipelineCard } from '~/stores/pipelineCards'
import type { PipelineStage } from '~/stores/pipelines'

const POSITION_GAP = 100

export function computeInsertPosition(
  list: PipelineCard[],
  index: number,
  excludeCardId?: number | string
): number {
  const filtered = excludeCardId !== undefined
    ? list.filter(c => String(c.id) !== String(excludeCardId))
    : [...list]
  if (filtered.length === 0) return POSITION_GAP
  const prev = filtered[index - 1]
  const next = filtered[index]
  if (!prev && !next) return POSITION_GAP
  if (!prev && next) return next.position / 2
  if (prev && !next) return prev.position + POSITION_GAP
  if (prev && next) return (prev.position + next.position) / 2
  return POSITION_GAP
}

export function computeStageInsertPosition(stages: PipelineStage[], index: number, excludeStageId?: number | string): number {
  const filtered = excludeStageId !== undefined
    ? stages.filter(s => String(s.id) !== String(excludeStageId))
    : [...stages]
  if (filtered.length === 0) return POSITION_GAP
  const prev = filtered[index - 1]
  const next = filtered[index]
  if (!prev && !next) return POSITION_GAP
  if (!prev && next) return next.position / 2
  if (prev && !next) return prev.position + POSITION_GAP
  if (prev && next) return (prev.position + next.position) / 2
  return POSITION_GAP
}
