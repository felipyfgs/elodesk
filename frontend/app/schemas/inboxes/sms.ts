import { z } from 'zod'

const bandwidthConfigSchema = z.object({
  accountId: z.string().min(1, 'Account ID is required'),
  applicationId: z.string().min(1, 'Application ID is required'),
  basicAuthUser: z.string().min(1, 'Username is required'),
  basicAuthPass: z.string().min(1, 'Password is required')
})

const zenviaConfigSchema = z.object({
  apiToken: z.string().min(1, 'API token is required')
})

export const smsStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  provider: z.enum(['bandwidth', 'zenvia']),
  phoneNumber: z.string().min(1, 'Phone number is required')
})

export const smsStepProviderSchema = z.object({
  bandwidth: bandwidthConfigSchema.optional(),
  zenvia: zenviaConfigSchema.optional()
}).refine(
  data => !!data.bandwidth || !!data.zenvia,
  { message: 'Provider configuration is required', path: ['provider'] }
)

export const smsInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  provider: z.enum(['bandwidth', 'zenvia']),
  phoneNumber: z.string().min(1, 'Phone number is required'),
  bandwidth: bandwidthConfigSchema.optional(),
  zenvia: zenviaConfigSchema.optional()
}).refine(
  (data) => {
    if (data.provider === 'bandwidth') return !!data.bandwidth
    if (data.provider === 'zenvia') return !!data.zenvia
    return false
  },
  { message: 'Provider configuration is required', path: ['provider'] }
)

export type SmsInboxForm = z.infer<typeof smsInboxSchema>
export type SmsStepSetup = z.infer<typeof smsStepSetupSchema>
export type SmsStepProvider = z.infer<typeof smsStepProviderSchema>
