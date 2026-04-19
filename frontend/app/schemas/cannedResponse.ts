import { z } from 'zod'

export const cannedResponseSchema = z.object({
  short_code: z.string().regex(/^[a-z0-9][a-z0-9_-]{0,31}$/, 'Código inválido').min(1),
  content: z.string().min(1, 'Conteúdo obrigatório').max(10000)
})

export type CannedResponseForm = z.infer<typeof cannedResponseSchema>
