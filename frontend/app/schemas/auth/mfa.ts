import { z } from 'zod/v4'

export const mfaCodeSchema = z.object({
  code: z.string().length(6, 'Código deve ter 6 dígitos')
})

export const mfaEnableSchema = z.object({
  code: z.string().length(6, 'Código deve ter 6 dígitos')
})

export const mfaVerifySchema = z.object({
  mfaToken: z.string().min(1, 'Token MFA obrigatório'),
  code: z.string().min(1, 'Código obrigatório')
})

export const mfaDisableSchema = z.object({
  currentPassword: z.string().min(1, 'Senha atual obrigatória')
})

export type MfaCodeForm = z.infer<typeof mfaCodeSchema>
export type MfaEnableForm = z.infer<typeof mfaEnableSchema>
export type MfaVerifyForm = z.infer<typeof mfaVerifySchema>
export type MfaDisableForm = z.infer<typeof mfaDisableSchema>
