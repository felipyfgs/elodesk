import { z } from 'zod/v4'

export const accountSettingsSchema = z.object({
  name: z.string().min(1, 'Nome obrigatório'),
  locale: z.enum(['pt-BR', 'en']),
  timezone: z.string().min(1, 'Fuso horário obrigatório')
})

export type AccountSettingsForm = z.infer<typeof accountSettingsSchema>
