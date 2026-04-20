import { z } from 'zod'

export const notificationPreferencesSchema = z.object({
  mentions: z.boolean().default(true),
  assignment: z.boolean().default(true),
  new_conversation: z.boolean().default(true),
  unread_message: z.boolean().default(false),
  sla_breach: z.boolean().default(true),
  email_enabled: z.boolean().default(false)
})

export type NotificationPreferences = z.infer<typeof notificationPreferencesSchema>
