import { z } from 'zod'

export const noteSchema = z.object({
  content: z.string().min(1, 'Conteúdo obrigatório').max(50000)
})

export type NoteForm = z.infer<typeof noteSchema>
