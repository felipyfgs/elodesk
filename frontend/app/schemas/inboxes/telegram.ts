import { z } from 'zod'

export const telegramStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required')
})

export const telegramStepCredentialsSchema = z.object({
  botToken: z.string().min(1, 'Bot token is required')
})

export const telegramInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  botToken: z.string().min(1, 'Bot token is required')
})

export type TelegramInboxForm = z.infer<typeof telegramInboxSchema>
export type TelegramStepSetup = z.infer<typeof telegramStepSetupSchema>
export type TelegramStepCredentials = z.infer<typeof telegramStepCredentialsSchema>
