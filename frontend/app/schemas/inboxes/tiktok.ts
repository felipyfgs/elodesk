import { z } from 'zod'

export const tiktokInboxSchema = z.object({
  name: z.string().min(1, 'Name is required')
})

export type TiktokInboxForm = z.infer<typeof tiktokInboxSchema>
