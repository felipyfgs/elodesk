import { z } from 'zod'

export const profileSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  avatarUrl: z.string().optional()
})

export type ProfileForm = z.infer<typeof profileSchema>

export const passwordChangeSchema = z.object({
  currentPassword: z.string().min(1),
  newPassword: z.string().min(8),
  confirmPassword: z.string().min(8)
}).refine(v => v.newPassword === v.confirmPassword, {
  path: ['confirmPassword'],
  message: 'passwords must match'
})

export type PasswordChangeForm = z.infer<typeof passwordChangeSchema>
