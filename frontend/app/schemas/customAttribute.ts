import { z } from 'zod'

export const customAttributeSchema = z.object({
  attribute_key: z.string().regex(/^[a-z][a-z0-9_]{0,62}$/, 'Chave inválida'),
  attribute_display_name: z.string().min(1, 'Nome de exibição obrigatório'),
  attribute_display_type: z.enum(['text', 'number', 'currency', 'percent', 'link', 'date', 'list', 'checkbox']),
  attribute_model: z.enum(['contact', 'conversation']),
  attribute_values: z.any().nullable().optional(),
  attribute_description: z.string().max(500).nullable().optional(),
  regex_pattern: z.string().nullable().optional(),
  default_value: z.string().nullable().optional()
})

export type CustomAttributeForm = z.infer<typeof customAttributeSchema>
