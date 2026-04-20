<script setup lang="ts">
import { formatTimeAgo } from '@vueuse/core'
import type { Inbox } from '~/stores/inboxes'

const props = defineProps<{
  inbox: Inbox
}>()

const { t } = useI18n()

const channelConfig: Record<string, { icon: string, color: string, bg: string }> = {
  api: { icon: 'i-lucide-webhook', color: 'text-blue-500', bg: 'bg-blue-500/10 ring-blue-500/25' },
  whatsapp: { icon: 'i-simple-icons-whatsapp', color: 'text-green-500', bg: 'bg-green-500/10 ring-green-500/25' },
  sms: { icon: 'i-lucide-message-square', color: 'text-purple-500', bg: 'bg-purple-500/10 ring-purple-500/25' },
  instagram: { icon: 'i-simple-icons-instagram', color: 'text-pink-500', bg: 'bg-pink-500/10 ring-pink-500/25' },
  facebook_page: { icon: 'i-simple-icons-facebook', color: 'text-blue-600', bg: 'bg-blue-600/10 ring-blue-600/25' },
  telegram: { icon: 'i-simple-icons-telegram', color: 'text-sky-500', bg: 'bg-sky-500/10 ring-sky-500/25' },
  web_widget: { icon: 'i-lucide-globe', color: 'text-amber-500', bg: 'bg-amber-500/10 ring-amber-500/25' },
  email: { icon: 'i-lucide-mail', color: 'text-orange-500', bg: 'bg-orange-500/10 ring-orange-500/25' }
}

const defaultConfig = { icon: 'i-lucide-webhook', color: 'text-blue-500', bg: 'bg-blue-500/10 ring-blue-500/25' }
const config = computed(() => channelConfig[props.inbox.channelType] ?? defaultConfig)

const badgeColor = computed(() => {
  const type = props.inbox.channelType
  if (type === 'whatsapp' || type === 'telegram') return 'success'
  if (type === 'api') return 'primary'
  return 'neutral'
})

const visibleAgents = computed(() => (props.inbox.agents ?? []).slice(0, 3))
const extraAgents = computed(() => Math.max(0, (props.inbox.agents?.length ?? 0) - 3))
</script>

<template>
  <UPageCard
    variant="subtle"
    :to="`/inboxes/${inbox.id}`"
    :ui="{
      container: 'gap-3',
      wrapper: 'items-start',
      leading: `p-2.5 rounded-full ring ring-inset ${config.bg}`
    }"
  >
    <template #leading>
      <UIcon :name="config.icon" :class="['size-5 shrink-0', config.color]" />
    </template>

    <template #title>
      <div class="flex items-center gap-2">
        <span class="font-medium">{{ inbox.name }}</span>
        <UBadge :color="badgeColor" variant="subtle" size="xs">
          {{ t(`inboxes.channels.${inbox.channelType}`) }}
        </UBadge>
      </div>
    </template>

    <template #description>
      <div class="flex items-center gap-3 text-xs text-muted">
        <span v-if="inbox.openConversationCount != null">
          {{ inbox.openConversationCount }} open
        </span>
        <span v-if="inbox.lastActivityAt">
          {{ formatTimeAgo(new Date(inbox.lastActivityAt)) }}
        </span>
      </div>
    </template>

    <div v-if="visibleAgents.length" class="flex items-center gap-1 mt-2">
      <UAvatarGroup size="xs" :max="3">
        <UTooltip v-for="agent in visibleAgents" :key="agent.userId" :text="agent.user?.name">
          <UAvatar
            :src="agent.user?.avatarUrl ?? undefined"
            :alt="agent.user?.name"
            size="xs"
          />
        </UTooltip>
      </UAvatarGroup>
      <span v-if="extraAgents > 0" class="text-xs text-muted ml-1">+{{ extraAgents }}</span>
    </div>
  </UPageCard>
</template>
