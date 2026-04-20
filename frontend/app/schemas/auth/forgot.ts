import { z } from 'zod/v4'

export const forgotSchema = z.object({
  email: z.email('Email inválido')
})

export type ForgotForm = z.infer<typeof forgotSchema>
