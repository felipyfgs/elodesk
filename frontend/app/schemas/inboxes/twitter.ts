import { z } from 'zod'

export const twitterInboxSchema = z.object({})

export type TwitterInboxForm = z.infer<typeof twitterInboxSchema>
