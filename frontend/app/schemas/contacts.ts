import { z } from 'zod'

const optionalString = (max: number) =>
  z.string().max(max).optional().or(z.literal(''))

const optionalUrl = (max: number, message = 'Invalid URL') =>
  z.string().max(max).url(message).optional().or(z.literal(''))

type Translator = (key: string) => string

export function createContactCreateSchema(t: Translator) {
  return z.object({
    first_name: z.string().min(1, t('contacts.form.firstNameRequired')).max(120),
    last_name: optionalString(120),
    email: z.string().email(t('contacts.form.emailInvalid')).optional().or(z.literal('')),
    phone_number: optionalString(30),
    city: optionalString(120),
    country: optionalString(120),
    bio: optionalString(500),
    company_name: optionalString(120),
    linkedin: optionalUrl(255, t('contacts.form.urlInvalid')),
    facebook: optionalUrl(255, t('contacts.form.urlInvalid')),
    instagram: optionalUrl(255, t('contacts.form.urlInvalid')),
    twitter: optionalUrl(255, t('contacts.form.urlInvalid')),
    github: optionalUrl(255, t('contacts.form.urlInvalid'))
  })
}

export const contactCreateSchema = z.object({
  first_name: z.string().min(1, 'First name is required').max(120),
  last_name: optionalString(120),
  email: z.string().email('Invalid email').optional().or(z.literal('')),
  phone_number: optionalString(30),
  city: optionalString(120),
  country: optionalString(120),
  bio: optionalString(500),
  company_name: optionalString(120),
  linkedin: optionalUrl(255),
  facebook: optionalUrl(255),
  instagram: optionalUrl(255),
  twitter: optionalUrl(255),
  github: optionalUrl(255)
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

export const contactMergeSchema = z.object({
  primary_contact_id: z.number().int().positive()
})

export const contactBlockSchema = z.object({
  blocked: z.boolean()
})

export const contactAvatarSchema = z.object({
  object_key: z.string().min(1)
})

export type ContactCreateForm = z.infer<typeof contactCreateSchema>
export type ContactUpdateForm = z.infer<typeof contactUpdateSchema>
export type ContactImportRow = z.infer<typeof contactImportRowSchema>
export type ContactSegmentForm = z.infer<typeof contactSegmentSchema>
export type ContactMergeForm = z.infer<typeof contactMergeSchema>
export type ContactBlockForm = z.infer<typeof contactBlockSchema>
export type ContactAvatarForm = z.infer<typeof contactAvatarSchema>
