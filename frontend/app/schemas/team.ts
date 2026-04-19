import { z } from 'zod'

export const teamSchema = z.object({
  name: z.string().min(1, 'Nome obrigatório').max(255),
  description: z.string().max(500).nullable().optional(),
  allow_auto_assign: z.boolean().default(false)
})

export type TeamForm = z.infer<typeof teamSchema>

export const addTeamMembersSchema = z.object({
  user_ids: z.array(z.number()).min(1, 'Selecione ao menos um membro')
})
