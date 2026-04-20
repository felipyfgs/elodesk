import { z } from 'zod'

export const instagramStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  instagramId: z.string().min(1, 'Instagram ID is required')
})

export const instagramStepCredentialsSchema = z.object({
  accessToken: z.string().min(1, 'Access token is required')
})

export const instagramInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  instagramId: z.string().min(1, 'Instagram ID is required'),
  accessToken: z.string().min(1, 'Access token is required')
})

export type InstagramInboxForm = z.infer<typeof instagramInboxSchema>
export type InstagramStepSetup = z.infer<typeof instagramStepSetupSchema>
export type InstagramStepCredentials = z.infer<typeof instagramStepCredentialsSchema>
