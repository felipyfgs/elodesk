import { z } from 'zod'

export const emailStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  provider: z.enum(['generic', 'google', 'microsoft']),
  email: z.string().email('Valid email is required')
})

export const emailStepImapSchema = z.object({
  imapAddress: z.string().min(1, 'IMAP address is required'),
  imapPort: z.number().int().positive('Port must be positive'),
  imapLogin: z.string().optional(),
  imapPassword: z.string().min(1, 'IMAP password is required'),
  imapEnableSsl: z.boolean()
})

export const emailStepSmtpSchema = z.object({
  smtpAddress: z.string().min(1, 'SMTP address is required'),
  smtpPort: z.number().int().positive('Port must be positive'),
  smtpLogin: z.string().optional(),
  smtpPassword: z.string().min(1, 'SMTP password is required'),
  smtpEnableSsl: z.boolean()
})

export const emailInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  provider: z.enum(['generic', 'google', 'microsoft']),
  email: z.string().email('Valid email is required'),
  imapAddress: z.string().optional(),
  imapPort: z.number().int().positive().optional(),
  imapLogin: z.string().optional(),
  imapPassword: z.string().optional(),
  imapEnableSsl: z.boolean().optional(),
  imapEnabled: z.boolean().optional(),
  smtpAddress: z.string().optional(),
  smtpPort: z.number().int().positive().optional(),
  smtpLogin: z.string().optional(),
  smtpPassword: z.string().optional(),
  smtpEnableSsl: z.boolean().optional()
}).refine(
  data => data.provider !== 'generic' || !!data.imapPassword,
  { message: 'IMAP password is required for generic provider', path: ['imapPassword'] }
)

export type EmailInboxForm = z.infer<typeof emailInboxSchema>
export type EmailStepSetup = z.infer<typeof emailStepSetupSchema>
export type EmailStepImap = z.infer<typeof emailStepImapSchema>
export type EmailStepSmtp = z.infer<typeof emailStepSmtpSchema>
