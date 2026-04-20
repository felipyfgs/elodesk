import { z } from 'zod/v4'

export const loginSchema = z.object({
  email: z.email('Email inválido'),
  password: z.string().min(1, 'Senha obrigatória')
})

export type LoginForm = z.infer<typeof loginSchema>

export const loginResponseSchema = z.union([
  z.object({
    success: z.literal(true),
    data: z.object({
      user: z.object({ id: z.number(), email: z.string(), name: z.string() }),
      account: z.object({ id: z.number(), name: z.string(), slug: z.string() }),
      accessToken: z.string(),
      refreshToken: z.string()
    })
  }),
  z.object({
    success: z.literal(true),
    data: z.object({
      mfaRequired: z.literal(true),
      mfaToken: z.string()
    })
  })
])

export type LoginResponse = z.infer<typeof loginResponseSchema>
