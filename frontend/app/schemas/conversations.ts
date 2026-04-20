import { z } from 'zod'

export const conversationFiltersSchema = z.object({
  tab: z.enum(['mine', 'unassigned', 'all', 'mentions'] as const).default('mine'),
  inbox_id: z.string().optional(),
  label_id: z.string().optional(),
  team_id: z.string().optional(),
  status: z.enum(['OPEN', 'PENDING', 'RESOLVED', 'SNOOZED'] as const).optional(),
  from: z.string().optional(),
  to: z.string().optional()
})

export const conversationBulkActionSchema = z.object({
  ids: z.array(z.string()).min(1),
  action: z.enum(['resolve', 'open', 'pending', 'snooze', 'assign_agent', 'assign_team', 'add_label', 'remove_label', 'mark_unread', 'delete'] as const),
  payload: z.record(z.string(), z.unknown()).optional()
})

export const conversationAssignSchema = z.object({
  assignee_id: z.string().nullable().optional(),
  team_id: z.string().nullable().optional()
})

export const conversationStatusSchema = z.object({
  status: z.enum(['OPEN', 'PENDING', 'RESOLVED', 'SNOOZED'] as const)
})

export const conversationSnoozeSchema = z.object({
  status: z.literal('SNOOZED'),
  snooze_until: z.string().optional()
})

export type ConversationFilters = z.infer<typeof conversationFiltersSchema>
export type ConversationBulkAction = z.infer<typeof conversationBulkActionSchema>
export type ConversationAssignForm = z.infer<typeof conversationAssignSchema>
export type ConversationStatusForm = z.infer<typeof conversationStatusSchema>
