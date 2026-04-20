import { z } from 'zod/v4'

export const resetSchema = z.object({
  password: z.string().min(8, 'Senha deve ter no mínimo 8 caracteres'),
  confirm: z.string().min(8, 'Senha deve ter no mínimo 8 caracteres')
}).refine(
  data => data.password === data.confirm,
  { path: ['confirm'], message: 'Senhas não conferem' }
)

export type ResetForm = z.infer<typeof resetSchema>
