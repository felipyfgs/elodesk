import { z } from 'zod'

export const savedFilterSchema = z.object({
  name: z.string().min(1, 'Nome obrigatório').max(255),
  filter_type: z.enum(['conversation', 'contact']),
  query: z.any()
})

export type SavedFilterForm = z.infer<typeof savedFilterSchema>

export const filterConditionSchema = z.object({
  attribute_key: z.string().min(1),
  filter_operator: z.enum([
    'equal_to', 'not_equal_to', 'contains', 'starts_with',
    'greater_than', 'less_than', 'in', 'between', 'is_null', 'is_not_null'
  ]),
  value: z.any().nullable()
})

export type FilterConditionForm = z.infer<typeof filterConditionSchema>
