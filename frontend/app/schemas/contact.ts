import { z } from 'zod'

export const contactSchema = z.object({
  name: z.string().min(1, 'Nome obrigatório').max(255).optional(),
  email: z.string().email('Email inválido').nullable().optional(),
  phone_number: z.string().max(30).nullable().optional(),
  identifier: z.string().max(255).nullable().optional()
})

export type ContactForm = z.infer<typeof contactSchema>
