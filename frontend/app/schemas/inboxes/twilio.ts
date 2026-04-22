import { z } from 'zod'

export const twilioStepMediumSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  medium: z.enum(['whatsapp', 'sms'])
})

export const twilioStepCredentialsSchema = z.object({
  accountSid: z.string().min(1, 'Account SID is required'),
  authToken: z.string().min(1, 'Auth token is required'),
  apiKeySid: z.string().optional().default('')
})

export const twilioStepSenderSchema = z.object({
  phoneNumber: z.string().optional().default(''),
  messagingServiceSid: z.string().optional().default('')
}).refine(
  data => Boolean(data.phoneNumber) !== Boolean(data.messagingServiceSid),
  { message: 'Provide exactly one of phone number or Messaging Service SID', path: ['phoneNumber'] }
)

export const twilioInboxSchema = z
  .object({
    name: z.string().min(1, 'Name is required'),
    medium: z.enum(['whatsapp', 'sms']),
    accountSid: z.string().min(1, 'Account SID is required'),
    authToken: z.string().min(1, 'Auth token is required'),
    apiKeySid: z.string().optional().default(''),
    phoneNumber: z.string().optional().default(''),
    messagingServiceSid: z.string().optional().default('')
  })
  .refine(
    data => Boolean(data.phoneNumber) !== Boolean(data.messagingServiceSid),
    { message: 'Provide exactly one of phone number or Messaging Service SID', path: ['phoneNumber'] }
  )

export type TwilioInboxForm = z.infer<typeof twilioInboxSchema>
export type TwilioStepMedium = z.infer<typeof twilioStepMediumSchema>
export type TwilioStepCredentials = z.infer<typeof twilioStepCredentialsSchema>
export type TwilioStepSender = z.infer<typeof twilioStepSenderSchema>
