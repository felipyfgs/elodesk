import { z } from 'zod'

export const lineStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required')
})

export const lineStepCredentialsSchema = z.object({
  lineChannelId: z.string().min(1, 'LINE channel ID is required'),
  lineChannelSecret: z.string().min(1, 'LINE channel secret is required'),
  lineChannelToken: z.string().min(1, 'LINE channel token is required')
})

export const lineInboxSchema = lineStepSetupSchema.merge(lineStepCredentialsSchema)

export type LineInboxForm = z.infer<typeof lineInboxSchema>
export type LineStepSetup = z.infer<typeof lineStepSetupSchema>
export type LineStepCredentials = z.infer<typeof lineStepCredentialsSchema>
