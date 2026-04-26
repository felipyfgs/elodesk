import type { AvatarProps } from '@nuxt/ui'

export interface Notification {
  id: string
  unread?: boolean
  sender: { name: string, avatar?: AvatarProps }
  body: string
  date: string
}

export interface Stat {
  title: string
  icon: string
  value: number | string
  variation?: number
  to?: string
  formatter?: (value: number) => string
}

export type Period = 'daily' | 'weekly' | 'monthly'

export interface Range {
  start: Date
  end: Date
}

export type ContactStatus = 'active' | 'inactive' | 'unknown'

export interface ContactRow {
  id: string
  name: string | null
  phoneNumber: string | null
  identifier: string | null
  email: string | null
  avatarUrl: string | null
  thumbnail: string | null
  status: ContactStatus
  createdAt: string
}
