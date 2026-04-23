const CHANNEL_ICONS: Record<string, string> = {
  api: 'i-lucide-webhook',
  whatsapp: 'i-simple-icons-whatsapp',
  sms: 'i-lucide-message-square',
  instagram: 'i-simple-icons-instagram',
  facebook_page: 'i-simple-icons-facebook',
  telegram: 'i-simple-icons-telegram',
  web_widget: 'i-lucide-globe',
  email: 'i-lucide-mail',
  line: 'i-simple-icons-line',
  tiktok: 'i-simple-icons-tiktok',
  twilio: 'i-lucide-cloud',
  twitter: 'i-simple-icons-x'
}

const CHANNEL_TYPE_ICONS: Record<string, string> = {
  'Channel::Api': 'i-lucide-webhook',
  'Channel::Whatsapp': 'i-simple-icons-whatsapp',
  'Channel::Twilio': 'i-lucide-cloud',
  'Channel::Sms': 'i-lucide-message-square',
  'Channel::Instagram': 'i-simple-icons-instagram',
  'Channel::FacebookPage': 'i-simple-icons-facebook',
  'Channel::Telegram': 'i-simple-icons-telegram',
  'Channel::Line': 'i-simple-icons-line',
  'Channel::Tiktok': 'i-simple-icons-tiktok',
  'Channel::WebWidget': 'i-lucide-globe',
  'Channel::Email': 'i-lucide-mail',
  'Channel::Twitter': 'i-simple-icons-x'
}

export function channelIcon(channelType: string): string {
  return CHANNEL_TYPE_ICONS[channelType] ?? CHANNEL_ICONS[channelType] ?? 'i-lucide-inbox'
}

export function channelKindIcon(kind: string): string {
  return CHANNEL_ICONS[kind] ?? 'i-lucide-inbox'
}
