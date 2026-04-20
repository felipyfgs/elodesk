import { z } from 'zod'

export const apiInboxSchema = z.object({
  name: z.string().min(1, 'Name is required')
})

export type ApiInboxForm = z.infer<typeof apiInboxSchema>
