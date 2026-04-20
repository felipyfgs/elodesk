import { z } from 'zod'

export const contactCreateSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  email: z.string().email('Invalid email').nullable().optional(),
  phone_number: z.string().max(30).nullable().optional(),
  identifier: z.string().max(255).nullable().optional()
})

export const contactUpdateSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  email: z.string().email('Invalid email').nullable().optional(),
  phone_number: z.string().max(30).nullable().optional(),
  identifier: z.string().max(255).nullable().optional()
})

export const contactImportRowSchema = z.object({
  name: z.string().optional(),
  email: z.string().email('Invalid email').nullable().optional(),
  phone: z.string().nullable().optional()
})

export const contactSegmentSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  filter_type: z.literal('contact'),
  query: z.object({
    operator: z.enum(['and', 'or']),
    conditions: z.array(z.object({
      attribute_key: z.string(),
      filter_operator: z.string(),
      value: z.string().nullable().optional()
    }))
  })
})

export type ContactCreateForm = z.infer<typeof contactCreateSchema>
export type ContactUpdateForm = z.infer<typeof contactUpdateSchema>
export type ContactImportRow = z.infer<typeof contactImportRowSchema>
export type ContactSegmentForm = z.infer<typeof contactSegmentSchema>
