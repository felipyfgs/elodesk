import { z } from 'zod'

export const slaSchema = z.object({
  name: z.string().min(1).max(255),
  firstResponseMinutes: z.number().int().min(1),
  resolutionMinutes: z.number().int().min(1),
  businessHoursOnly: z.boolean().default(false),
  inboxIds: z.array(z.number()).default([]),
  labelIds: z.array(z.number()).default([])
})

export type SlaForm = z.infer<typeof slaSchema>
