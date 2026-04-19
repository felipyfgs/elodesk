import { z } from 'zod'

export const labelSchema = z.object({
  title: z.string().min(1, 'Título obrigatório').max(255),
  color: z.string().regex(/^#[0-9A-Fa-f]{6}$/, 'Cor hex inválida').default('#1f93ff'),
  description: z.string().max(500).nullable().optional(),
  show_on_sidebar: z.boolean().default(false)
})

export type LabelForm = z.infer<typeof labelSchema>

export const applyLabelSchema = z.object({
  label_id: z.number()
})
