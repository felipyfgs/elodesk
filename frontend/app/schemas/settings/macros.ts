import { z } from 'zod'

export const macroActionSchema = z.object({
  name: z.enum([
    'assign_agent', 'assign_team', 'add_label', 'remove_label',
    'change_status', 'snooze_until', 'send_message', 'add_note'
  ]),
  params: z.record(z.string(), z.any()).optional()
})

export type MacroAction = z.infer<typeof macroActionSchema>

export const macroSchema = z.object({
  name: z.string().min(1).max(255),
  visibility: z.enum(['account', 'personal']).default('account'),
  conditions: z.record(z.string(), z.any()).default({}),
  actions: z.array(macroActionSchema).default([])
})

export type MacroForm = z.infer<typeof macroSchema>
