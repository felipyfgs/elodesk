import { z } from 'zod'

export const webWidgetStepSetupSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  websiteUrl: z.string().url('Valid URL is required')
})

export const webWidgetStepAppearanceSchema = z.object({
  widgetColor: z.string().optional(),
  welcomeTitle: z.string().optional(),
  welcomeTagline: z.string().optional(),
  replyTime: z.enum(['in_a_few_minutes', 'in_a_few_hours', 'in_a_day']).optional()
})

export const webWidgetInboxSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  websiteUrl: z.string().url('Valid URL is required'),
  widgetColor: z.string().optional(),
  welcomeTitle: z.string().optional(),
  welcomeTagline: z.string().optional(),
  replyTime: z.enum(['in_a_few_minutes', 'in_a_few_hours', 'in_a_day']).optional()
})

export type WebWidgetInboxForm = z.infer<typeof webWidgetInboxSchema>
export type WebWidgetStepSetup = z.infer<typeof webWidgetStepSetupSchema>
export type WebWidgetStepAppearance = z.infer<typeof webWidgetStepAppearanceSchema>
