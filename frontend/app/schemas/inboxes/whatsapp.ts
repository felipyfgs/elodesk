import { z } from 'zod'

export const whatsappStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  provider: z.enum(['whatsapp_cloud', 'default_360dialog']),
  phoneNumber: z.string().min(1, 'Phone number is required')
})

export const whatsappStepCredentialsSchema = z.object({
  phoneNumberId: z.string().optional(),
  businessAccountId: z.string().optional(),
  apiKey: z.string().min(1, 'API key is required')
})

export const whatsappInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  provider: z.enum(['whatsapp_cloud', 'default_360dialog']),
  phoneNumber: z.string().min(1, 'Phone number is required'),
  phoneNumberId: z.string().optional(),
  businessAccountId: z.string().optional(),
  apiKey: z.string().min(1, 'API key is required')
})

export type WhatsAppInboxForm = z.infer<typeof whatsappInboxSchema>
export type WhatsAppStepSetup = z.infer<typeof whatsappStepSetupSchema>
export type WhatsAppStepCredentials = z.infer<typeof whatsappStepCredentialsSchema>
