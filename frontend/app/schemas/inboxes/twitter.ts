import { z } from 'zod'

// Twitter provisioning is OAuth 1.0a redirect-based — no local form state.
export const twitterInboxSchema = z.object({})

export type TwitterInboxForm = z.infer<typeof twitterInboxSchema>
