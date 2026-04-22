import { defineStore } from 'pinia'

export interface InboxAgent {
  id: string
  inboxId: string
  userId: string
  user?: { id: string, name: string, avatarUrl?: string | null }
  createdAt: string
}

export interface ChannelWhatsAppData {
  provider: string
  phoneNumber: string
  phoneNumberId?: string | null
  businessAccountId?: string | null
  messageTemplatesSyncedAt?: string | null
}

export interface ChannelApiData {
  id?: string | number
  identifier: string
  webhookUrl: string
  hmacMandatory: boolean
  additionalAttributes?: Record<string, unknown>
  createdAt?: string
  updatedAt?: string
}

export interface ChannelWebWidgetData {
  id: string | number
  websiteToken: string
  websiteUrl: string
  widgetColor: string
  welcomeTitle: string
  welcomeTagline: string
  replyTime: string
  featureFlags?: string
  embedScript: string
  createdAt: string
  updatedAt: string
}

export interface ChannelEmailData {
  email: string
  provider: string
  imapAddress?: string | null
  imapPort?: number | null
  imapLogin?: string | null
  imapEnableSsl: boolean
  imapEnabled: boolean
  smtpAddress?: string | null
  smtpPort?: number | null
  smtpLogin?: string | null
  smtpEnableSsl: boolean
  verifiedForSending: boolean
  requiresReauth: boolean
  emailCreatedAt: string
}

export interface ChannelLineData {
  id: string | number
  lineChannelId: string
  botBasicId?: string | null
  botDisplayName?: string | null
  requiresReauth: boolean
  createdAt: string
  updatedAt: string
}

export interface ChannelTiktokData {
  id: string | number
  businessId: string
  displayName?: string | null
  username?: string | null
  expiresAt: string
  refreshTokenExpiresAt: string
  requiresReauth: boolean
  createdAt: string
  updatedAt: string
}

export interface ChannelTwilioData {
  id: string | number
  medium: string
  accountSid: string
  apiKeySid?: string | null
  phoneNumber?: string | null
  messagingServiceSid?: string | null
  webhookIdentifier: string
  contentTemplatesLastUpdated?: string | null
  requiresReauth: boolean
  createdAt: string
  updatedAt: string
}

export interface ChannelTwitterData {
  id: string | number
  profileId: string
  screenName?: string | null
  tweetsEnabled: boolean
  requiresReauth: boolean
  createdAt: string
  updatedAt: string
}

export interface Inbox {
  id: string
  accountId: string
  channelId: string
  name: string
  channelType: string
  createdAt: string
  updatedAt?: string
  channelApi?: {
    id?: string | number
    identifier: string
    webhookUrl: string
    hmacMandatory: boolean
    additionalAttributes?: Record<string, unknown>
  } | null
  channelWhatsApp?: ChannelWhatsAppData | null
  channelWebWidget?: ChannelWebWidgetData | null
  channelEmail?: ChannelEmailData | null
  channelLine?: ChannelLineData | null
  channelTiktok?: ChannelTiktokData | null
  channelTwilio?: ChannelTwilioData | null
  channelTwitter?: ChannelTwitterData | null
  agents?: InboxAgent[]
  openConversationCount?: number
  lastActivityAt?: string
}

export const useInboxesStore = defineStore('inboxes', {
  state: () => ({
    list: [] as Inbox[],
    loading: false
  }),
  actions: {
    setAll(list: Inbox[]) {
      this.list = list
    },
    upsert(inbox: Inbox) {
      const idx = this.list.findIndex(i => i.id === inbox.id)
      if (idx >= 0) this.list[idx] = inbox
      else this.list.push(inbox)
    },
    remove(id: string) {
      this.list = this.list.filter(i => i.id !== id)
    }
  }
})
