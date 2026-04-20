import { z } from 'zod'

export const agentInviteSchema = z.object({
  email: z.string().email(),
  role: z.enum(['agent', 'admin', 'owner']).default('agent'),
  name: z.string().optional()
})

export type AgentInviteForm = z.infer<typeof agentInviteSchema>

export const agentUpdateSchema = z.object({
  role: z.enum(['agent', 'admin', 'owner']).optional(),
  status: z.string().optional()
})

export type AgentUpdateForm = z.infer<typeof agentUpdateSchema>
