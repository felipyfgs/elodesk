import { z } from 'zod'

export const facebookPageStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  pageId: z.string().min(1, 'Page ID is required')
})

export const facebookPageStepCredentialsSchema = z.object({
  pageAccessToken: z.string().min(1, 'Page access token is required'),
  userAccessToken: z.string().optional(),
  instagramId: z.string().optional()
})

export const facebookPageInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  pageId: z.string().min(1, 'Page ID is required'),
  pageAccessToken: z.string().min(1, 'Page access token is required'),
  userAccessToken: z.string().optional(),
  instagramId: z.string().optional()
})

export type FacebookPageInboxForm = z.infer<typeof facebookPageInboxSchema>
export type FacebookPageStepSetup = z.infer<typeof facebookPageStepSetupSchema>
export type FacebookPageStepCredentials = z.infer<typeof facebookPageStepCredentialsSchema>
