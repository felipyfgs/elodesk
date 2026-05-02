import { z } from 'zod'

export const cardKindEnum = z.enum(['free', 'contact', 'conversation'])
export type CardKind = z.infer<typeof cardKindEnum>

export const linkedEntityTypeEnum = z.enum(['contact', 'conversation'])
export type LinkedEntityType = z.infer<typeof linkedEntityTypeEnum>

export const terminalKindEnum = z.enum(['won', 'lost', 'resolved'])
export type TerminalKind = z.infer<typeof terminalKindEnum>

const hexColor = z.string().regex(/^#[0-9A-Fa-f]{6}$/, 'Cor hex inválida')

export const stageTemplateSchema = z.object({
  name: z.string().min(1).max(255),
  color: hexColor,
  is_terminal: z.boolean().optional(),
  terminal_kind: terminalKindEnum.optional()
})
export type StageTemplate = z.infer<typeof stageTemplateSchema>

export const pipelineTemplateSchema = z.object({
  key: z.string(),
  name: z.string(),
  description: z.string(),
  icon: z.string(),
  color: z.string(),
  card_kind: cardKindEnum,
  stages: z.array(stageTemplateSchema)
})
export type PipelineTemplate = z.infer<typeof pipelineTemplateSchema>

export const stageSchema = z.object({
  id: z.union([z.string(), z.number()]),
  pipelineId: z.union([z.string(), z.number()]),
  name: z.string(),
  position: z.number(),
  color: z.string(),
  isTerminal: z.boolean(),
  terminalKind: terminalKindEnum.nullish(),
  createdAt: z.string(),
  updatedAt: z.string()
})
export type Stage = z.infer<typeof stageSchema>

export const pipelineSchema = z.object({
  id: z.union([z.string(), z.number()]),
  accountId: z.union([z.string(), z.number()]),
  name: z.string(),
  description: z.string().nullish(),
  templateKey: z.string().nullish(),
  cardKind: cardKindEnum,
  icon: z.string().nullish(),
  color: z.string(),
  archivedAt: z.string().nullish(),
  createdBy: z.union([z.string(), z.number()]).nullish(),
  createdAt: z.string(),
  updatedAt: z.string(),
  stages: z.array(stageSchema).optional()
})
export type Pipeline = z.infer<typeof pipelineSchema>

export const cardSchema = z.object({
  id: z.union([z.string(), z.number()]),
  pipelineId: z.union([z.string(), z.number()]),
  stageId: z.union([z.string(), z.number()]),
  position: z.number(),
  title: z.string(),
  description: z.string().nullish(),
  valueCents: z.number().nullish(),
  valueCurrency: z.string().nullish(),
  dueDate: z.string().nullish(),
  customAttrs: z.record(z.string(), z.unknown()).default({}),
  linkedEntityType: linkedEntityTypeEnum.nullish(),
  linkedEntityId: z.union([z.string(), z.number()]).nullish(),
  assigneeUserIds: z.array(z.union([z.string(), z.number()])).default([]),
  labelIds: z.array(z.union([z.string(), z.number()])).default([]),
  createdBy: z.union([z.string(), z.number()]).nullish(),
  createdAt: z.string(),
  updatedAt: z.string()
})
export type Card = z.infer<typeof cardSchema>

export const createPipelineSchema = z.object({
  name: z.string().min(1, 'Nome obrigatório').max(255),
  description: z.string().max(2000).optional(),
  template_key: z.string().optional(),
  card_kind: cardKindEnum.optional(),
  icon: z.string().optional(),
  color: hexColor.optional(),
  stages: z.array(stageTemplateSchema).optional()
})
export type CreatePipelineForm = z.infer<typeof createPipelineSchema>

export const updatePipelineSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  description: z.string().max(2000).nullable().optional(),
  icon: z.string().optional(),
  color: hexColor.optional()
})
export type UpdatePipelineForm = z.infer<typeof updatePipelineSchema>

export const createStageSchema = z.object({
  name: z.string().min(1, 'Nome obrigatório').max(255),
  color: hexColor.optional(),
  is_terminal: z.boolean().optional(),
  terminal_kind: terminalKindEnum.optional()
})
export type CreateStageForm = z.infer<typeof createStageSchema>

export const updateStageSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  color: hexColor.optional(),
  position: z.number().optional(),
  is_terminal: z.boolean().optional(),
  terminal_kind: terminalKindEnum.optional()
})
export type UpdateStageForm = z.infer<typeof updateStageSchema>

const dateOnly = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Data inválida (YYYY-MM-DD)')

export const createCardSchema = z.object({
  stage_id: z.union([z.string(), z.number()]),
  title: z.string().min(1, 'Título obrigatório').max(500),
  description: z.string().optional(),
  value_cents: z.number().int().optional(),
  value_currency: z.string().length(3).optional(),
  due_date: dateOnly.optional(),
  custom_attrs: z.record(z.string(), z.unknown()).optional(),
  linked_entity_type: linkedEntityTypeEnum.optional(),
  linked_entity_id: z.union([z.string(), z.number()]).optional()
})
export type CreateCardForm = z.infer<typeof createCardSchema>

export const updateCardSchema = z.object({
  title: z.string().min(1).max(500).optional(),
  description: z.string().nullable().optional(),
  value_cents: z.number().int().nullable().optional(),
  value_currency: z.string().length(3).nullable().optional(),
  due_date: dateOnly.nullable().optional(),
  custom_attrs: z.record(z.string(), z.unknown()).optional()
})
export type UpdateCardForm = z.infer<typeof updateCardSchema>

export const moveCardSchema = z.object({
  stage_id: z.union([z.string(), z.number()]),
  position: z.number()
})
export type MoveCardForm = z.infer<typeof moveCardSchema>
